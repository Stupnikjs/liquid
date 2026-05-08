package swap

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/connector"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

var UniswapFees = []uint32{100, 500, 3000, 10000}

type QuoteExactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	AmountIn          *big.Int
	Fee               *big.Int
	SqrtPriceLimitX96 *big.Int
}

// Returns false if no pool
type SingleQuoterFunc func(
	conn *connector.Connector,
	marketp morpho.MarketParams,
	uniswapQuoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
	fee uint32,
) (PoolEdge, bool)

type Quoter struct {
	quoteSingle SingleQuoterFunc
}

func NewQuoter() *Quoter {
	return &Quoter{quoteSingle: RPCQuoteSingle}
}

// inject a quoter func
func NewQuoterWithFunc(fn SingleQuoterFunc) *Quoter {
	return &Quoter{quoteSingle: fn}
}

func Quote(conn *connector.Connector, marketp morpho.MarketParams, uniswapQuoterAddr common.Address, amountIn, oraclePrice *big.Int) (PoolEdge, bool) {
	return NewQuoter().Quote(conn, marketp, uniswapQuoterAddr, amountIn, oraclePrice)
}

func QuoteBinarySearch(conn *connector.Connector, marketp morpho.MarketParams, uniswapQuoterAddr common.Address, amountIn, oraclePrice *big.Int) (PoolEdge, bool) {
	return NewQuoter().QuoteBinarySearch(conn, marketp, uniswapQuoterAddr, amountIn, oraclePrice)
}

// binary search way more effective
func (q *Quoter) Quote(
	conn *connector.Connector,
	marketp morpho.MarketParams,
	uniswapQuoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
) (PoolEdge, bool) {
	maxSlippage := MaxSlippage(marketp.LLTV)
	current := new(big.Int).Set(amountIn)

	for current.Sign() > 0 {
		best, ok := q.bestUniFeeTier(conn, marketp, uniswapQuoterAddr, current, oraclePrice, maxSlippage, 200*time.Millisecond)
		if ok {
			fmt.Printf("Pair %s/%s | source: uniswap | fee: %d | slippage: %.4f%%\n",
				marketp.CollateralTokenStr, marketp.LoanTokenStr, best.Fee, best.WCSlippage)
			return best, true
		}
		current.Div(current, big.NewInt(4))
	}

	return PoolEdge{}, false
}

// iterate over mid amount to swap to find best slippage for max amount
func (q *Quoter) QuoteBinarySearch(
	conn *connector.Connector,
	marketp morpho.MarketParams,
	uniswapQuoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
) (PoolEdge, bool) {
	maxSlippage := MaxSlippage(marketp.LLTV)

	lo := big.NewInt(1)
	hi := new(big.Int).Set(amountIn)
	var best PoolEdge
	found := false

	for i := 0; i < 12 && lo.Cmp(hi) <= 0; i++ {
		mid := new(big.Int).Rsh(new(big.Int).Add(lo, hi), 1)

		result, ok := q.bestUniFeeTier(conn, marketp, uniswapQuoterAddr, mid, oraclePrice, maxSlippage, 400*time.Millisecond)
		if ok {
			best = result
			found = true
			lo = new(big.Int).Add(mid, big.NewInt(1))
		} else {
			hi = new(big.Int).Sub(mid, big.NewInt(1))
		}
	}

	if !found {
		fmt.Printf("no acceptable slippage found for %s -> %s\n",
			marketp.CollateralTokenStr, marketp.LoanTokenStr)
		return PoolEdge{}, false
	}

	fmt.Printf("acceptable slippage found for %s -> %s  %.4f%%\n",
		marketp.CollateralTokenStr, marketp.LoanTokenStr, best.WCSlippage)
	return best, true
}

func (q *Quoter) bestUniFeeTier(
	conn *connector.Connector,
	marketp morpho.MarketParams,
	uniswapQuoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
	maxSlippage float64,
	rateLimit time.Duration,
) (PoolEdge, bool) {
	var best PoolEdge
	found := false

	for _, fee := range UniswapFees {
		time.Sleep(rateLimit)
		// single call with fee
		result, ok := q.quoteSingle(conn, marketp, uniswapQuoterAddr, amountIn, oraclePrice, fee)
		if !ok {
			continue
		}
		if result.WCSlippage <= maxSlippage {
			if !found || result.WCAmountOut.Cmp(best.WCAmountOut) > 0 {
				best = result
				found = true
			}
		}
	}
	return best, found
}

// ---------------------------------------------------------------------------
// Implémentation RPC réelle
// ---------------------------------------------------------------------------

func RPCQuoteSingle(
	conn *connector.Connector,
	marketp morpho.MarketParams,
	uniswapQuoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
	fee uint32,
) (PoolEdge, bool) {
	params := QuoteExactInputSingleParams{
		TokenIn:           marketp.CollateralToken,
		TokenOut:          marketp.LoanToken,
		AmountIn:          amountIn,
		Fee:               big.NewInt(int64(fee)),
		SqrtPriceLimitX96: big.NewInt(0),
	}

	var (
		amountOut               *big.Int
		sqrtPriceX96After       *big.Int
		initializedTicksCrossed uint32
		gasEstimate             *big.Int
	)

	if err := conn.EthCallCtx(context.Background(),
		[]w3types.RPCCaller{eth.CallFunc(uniswapQuoterAddr, FuncQuoteExactInputSingleV2, params).Returns(
			&amountOut,
			&sqrtPriceX96After,
			&initializedTicksCrossed,
			&gasEstimate,
		)},
	); err != nil {
		return PoolEdge{}, false
	}

	return PoolEdge{
		TokenIn:      marketp.CollateralToken,
		TokenOut:     marketp.LoanToken,
		Router:       uniswapQuoterAddr,
		Fee:          fee,
		WCSlippage:   ComputeSlippage(amountIn, amountOut, oraclePrice),
		WCAmountIn:   new(big.Int).Set(amountIn),
		WCAmountOut:  amountOut,
		CalibratedAt: time.Now(),
	}, true
}

// ---------------------------------------------------------------------------
// Helpers slippage
// ---------------------------------------------------------------------------

func MaxSlippage(lltv *big.Int) float64 {
	lltvF, _ := new(big.Float).SetInt(lltv).Float64()
	lltvPct := lltvF / 1e18 * 100
	bonus := 100 - lltvPct
	const gasCushion = 0.1
	return bonus - gasCushion
}

func ComputeSlippage(amountIn, amountOut *big.Int, oraclePrice *big.Int) float64 {
	if oraclePrice == nil || oraclePrice.Sign() == 0 {
		return 0
	}

	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	expectedOut := new(big.Int).Div(
		new(big.Int).Mul(amountIn, oraclePrice),
		scale,
	)

	if expectedOut.Sign() == 0 {
		return 0
	}

	diff := new(big.Int).Sub(expectedOut, amountOut)
	slip, _ := new(big.Float).Quo(
		new(big.Float).SetInt(diff),
		new(big.Float).SetInt(expectedOut),
	).Float64()

	return slip * 100
}
