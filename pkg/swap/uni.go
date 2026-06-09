package swap

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

type UniSwapABI struct {
}

type QuoteExactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	AmountIn          *big.Int
	Fee               *big.Int
	SqrtPriceLimitX96 *big.Int
}

func (u *UniswapV3) BestAmountIn(
	conn connector.Connector,
	marketp MorphoMarket,
	amountIn *big.Int,
	oraclePrice *big.Int,
	rateLimit time.Duration,
) (PoolEdge, bool) {

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
		return PoolEdge{}, false
	}
	fmt.Printf("acceptable slippage found for %s  %.4f%%\n", marketp.GetPair(), best.WCSlippage)
	return best, true
}

func (u *UniswapV3) UniQuote(
	conn connector.Connector,
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
		result, ok := uniQuoteCall(conn, m, u.quoterAddr, u.routerAddr, amountIn, oraclePrice, fee, u.name)
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
	conn connector.Connector,
	marketp MorphoMarket,
	quoterAddr common.Address,
	routerAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	fee uint32,
	dexname string,
) (PoolEdge, bool) {
	params := QuoteExactInputSingleParams{
		TokenIn:           marketp.GetCollateralToken(),
		TokenOut:          marketp.GetLoanToken(),
		AmountIn:          amountIn,
		Fee:               big.NewInt(int64(fee)),
		SqrtPriceLimitX96: big.NewInt(0),
	}
	res := newResult()
	QuoteExactSingleInputMethod, err := loadQuoteExactInputSingleMethod()
	if err != nil {
		return PoolEdge{}, false
	}
	quoteCall, err := QuoteExactInputSingleCall(quoterAddr, params, res, &QuoteExactSingleInputMethod)
	if err != nil {
		return PoolEdge{}, false
	}
	err = conn.CallCtx(context.Background(), []rpc.BatchElem{quoteCall.Elem})
	if err != nil {
		return PoolEdge{}, false
	}
	err = quoteCall.Run()
	if err != nil {
		return PoolEdge{}, false
	}

	return PoolEdge{
		TokenIn:      marketp.GetCollateralToken(),
		TokenOut:     marketp.GetLoanToken(),
		Quoter:       quoterAddr,
		Router:       routerAddr,
		Fee:          fee,
		WCSlippage:   ComputeSlippage(amountIn, res.AmountOut, oraclePrice),
		WCAmountIn:   new(big.Int).Set(amountIn),
		WCAmountOut:  res.AmountOut,
		CalibratedAt: time.Now(),
		PriceAtQuote: oraclePrice,
		DexName:      dexname,
	}, true
}

func NewUniDex(
	quoterAddr common.Address,
	routerAddr common.Address,
	rateLimit time.Duration,
	name string,
) *UniswapV3 {
	return &UniswapV3{
		quoterAddr: quoterAddr,
		routerAddr: routerAddr,
		fees:       []uint32{100, 500, 3000, 10000},
		rateLimit:  rateLimit,
		name:       name,
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
	name       string
}

func (u *UniswapV3) DEX() string                   { return u.name }
func (u *UniswapV3) QuoterAddress() common.Address { return u.quoterAddr }
func (u *UniswapV3) RouterAddress() common.Address { return u.routerAddr }

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
