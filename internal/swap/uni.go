package swap

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
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

type Quoter struct {
	QuoterFunc lqtypes.QuoterFunc
}

func NewQuoter(fn lqtypes.QuoterFunc) *Quoter {
	return &Quoter{
		QuoterFunc: fn,
	}
}

// ---------------------------------------------------------------------------
// Binary search
// ---------------------------------------------------------------------------

// satisfy SingleQuote Interface
func (q *Quoter) QuoteBinarySearch(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
) (lqtypes.PoolEdge, bool) {
	maxSlippage := MaxSlippage(marketp.GetLLTV())
	fmt.Printf("max slippage for %s is: %f\n", marketp.GetPair(), maxSlippage)

	lo := big.NewInt(1)
	hi := new(big.Int).Set(amountIn)
	var best lqtypes.PoolEdge
	found := false

	for i := 0; i < 12 && lo.Cmp(hi) <= 0; i++ {
		mid := new(big.Int).Rsh(new(big.Int).Add(lo, hi), 1)

		result, ok := q.bestFeeTier(conn, marketp, quoterAddr, mid, oraclePrice, maxSlippage, 400*time.Millisecond, UniQuoteSingle)
		if ok {
			best = result
			found = true
			lo = new(big.Int).Add(mid, big.NewInt(1))
		} else {
			hi = new(big.Int).Sub(mid, big.NewInt(1))
		}
	}

	if !found {
		fmt.Printf("no acceptable slippage found for %s\n", marketp.GetPair())
		return lqtypes.PoolEdge{}, false
	}

	fmt.Printf("acceptable slippage found for %s  %.4f%%\n", marketp.GetPair(), best.WCSlippage)
	return best, true
}

func (q *Quoter) bestFeeTier(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	maxSlippage float64,
	rateLimit time.Duration,
	quoteSingle lqtypes.QuotSingleFunc,
) (lqtypes.PoolEdge, bool) {
	var best lqtypes.PoolEdge
	found := false

	for _, fee := range UniswapFees {
		time.Sleep(rateLimit)
		result, ok := UniQuoteSingle(conn, marketp, quoterAddr, amountIn, oraclePrice, fee)
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
// Implémentations RPC — utilisent les interfaces
// ---------------------------------------------------------------------------

func UniQuoteSingle(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	fee uint32,
) (lqtypes.PoolEdge, bool) {
	params := QuoteExactInputSingleParams{
		TokenIn:           marketp.GetCollateralToken(),
		TokenOut:          marketp.GetLoanToken(),
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
		[]w3types.RPCCaller{eth.CallFunc(quoterAddr, FuncQuoteExactInputSingleV2, params).Returns(
			&amountOut,
			&sqrtPriceX96After,
			&initializedTicksCrossed,
			&gasEstimate,
		)},
	); err != nil {
		return lqtypes.PoolEdge{}, false
	}

	return lqtypes.PoolEdge{
		TokenIn:      marketp.GetCollateralToken(),
		TokenOut:     marketp.GetLoanToken(),
		Router:       quoterAddr,
		Fee:          fee,
		WCSlippage:   ComputeSlippage(amountIn, amountOut, oraclePrice),
		WCAmountIn:   new(big.Int).Set(amountIn),
		WCAmountOut:  amountOut,
		CalibratedAt: time.Now(),
	}, true
}

func AerodromeQuoteSingle(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	fee uint32,
) (lqtypes.PoolEdge, bool) {
	return lqtypes.PoolEdge{}, false
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

func ComputeSlippage(amountIn, amountOut, oraclePrice *big.Int) float64 {
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
