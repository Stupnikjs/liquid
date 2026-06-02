package swap

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

type QuoteExactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	AmountIn          *big.Int
	Fee               *big.Int
	SqrtPriceLimitX96 *big.Int
}

func (u *UniswapV3) BestAmountIn(
	conn RPCClient,
	marketp MorphoMarket,
	amountIn *big.Int,
	oraclePrice *big.Int,
) (PoolEdge, bool) {
	maxSlippage := marketp.MaxSlippage()
	fmt.Printf("max slippage for %s is: %f\n", marketp.GetPair(), maxSlippage)

	lo := big.NewInt(1)
	hi := new(big.Int).Set(amountIn)
	var best PoolEdge
	found := false

	for i := 0; i < 12 && lo.Cmp(hi) <= 0; i++ {
		mid := new(big.Int).Rsh(new(big.Int).Add(lo, hi), 1)

		result, ok := u.UniQuote(conn, marketp, mid, oraclePrice)
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
		return PoolEdge{}, false
	}

	fmt.Printf("acceptable slippage found for %s  %.4f%%\n", marketp.GetPair(), best.WCSlippage)
	return best, true
}

func (u *UniswapV3) UniQuote(
	conn RPCClient,
	marketp MorphoMarket,
	amountIn *big.Int,
	oraclePrice *big.Int,
) (PoolEdge, bool) {
	var best PoolEdge
	var UniswapFees = []uint32{100, 500, 3000, 10000}
	found := false
	m := marketp
	for _, fee := range UniswapFees {
		time.Sleep(u.rateLimit)
		result, ok := uniQuoteCall(conn, m, u.quoterAddr, amountIn, oraclePrice, fee)
		if !ok {
			continue
		}

		if result.WCSlippage <= m.MaxSlippage() {
			if !found || result.WCAmountOut.Cmp(best.WCAmountOut) > 0 {
				best = result
				found = true
			}
		}
	}
	best.AmountInOffset = 164 // amountIn offset uniswap
	return best, found
}

// ---------------------------------------------------------------------------
// Implémentations RPC — utilisent les interfaces
// ---------------------------------------------------------------------------

func uniQuoteCall(
	conn RPCClient,
	marketp MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	fee uint32,
) (PoolEdge, bool) {
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

	if err := conn.CallCtx(context.Background(),
		[]w3types.RPCCaller{eth.CallFunc(quoterAddr, FuncQuoteExactInputSingleV2, params).Returns(
			&amountOut,
			&sqrtPriceX96After,
			&initializedTicksCrossed,
			&gasEstimate,
		)}...); err != nil {
		return PoolEdge{}, false
	}

	return PoolEdge{
		TokenIn:      marketp.GetCollateralToken(),
		TokenOut:     marketp.GetLoanToken(),
		Quoter:       quoterAddr,
		Fee:          fee,
		WCSlippage:   ComputeSlippage(amountIn, amountOut, oraclePrice),
		WCAmountIn:   new(big.Int).Set(amountIn),
		WCAmountOut:  amountOut,
		CalibratedAt: time.Now(),
		PriceAtQuote: oraclePrice,
	}, true
}

func NewUniDex(
	quoterAddr common.Address,
	routerAddr common.Address,
	rateLimit time.Duration,
) *UniswapV3 {
	return &UniswapV3{
		quoterAddr: quoterAddr,
		routerAddr: routerAddr,
		fees:       []uint32{100, 500, 3000, 10000},
		rateLimit:  rateLimit,
	}
}

// ---------------------------------------------------------------------------
// Uniswap V3 Quoter
// ---------------------------------------------------------------------------

// UniswapQuoter implements Quoter for Uniswap V3.
// It tries all standard fee tiers and returns the best amountOut.
type UniswapV3 struct {
	market     MorphoMarket
	quoterAddr common.Address
	routerAddr common.Address
	fees       []uint32
	rateLimit  time.Duration
}

func NewUniswapQuoter(
	market MorphoMarket,
	quoterAddr common.Address,
	routerAddr common.Address,
	oraclePrice *big.Int,
	rateLimit time.Duration,
) *UniswapV3 {
	return &UniswapV3{
		market:     market,
		quoterAddr: quoterAddr,
		routerAddr: routerAddr,
		fees:       []uint32{100, 500, 3000, 10000},
		rateLimit:  rateLimit,
	}
}

func (u *UniswapV3) DEX() string                   { return "uniswap-v3" }
func (u *UniswapV3) QuoterAddress() common.Address { return u.quoterAddr }
func (u *UniswapV3) RouterAddress() common.Address { return u.routerAddr }
