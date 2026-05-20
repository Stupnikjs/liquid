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

// Aerodrome CL fee tiers (en pips, équivalent Uniswap V3)
// 1 = 0.01%, 5 = 0.05%, 30 = 0.3%, 100 = 1%, 200 = 2%, 2000 = 20%
var AerodromeFees = []uint32{1, 5, 30, 100, 200, 2000}

func AeroQuoter(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	rateLimit time.Duration,
) (lqtypes.PoolEdge, bool) {
	maxSlippage := marketp.MaxSlippage()
	fmt.Printf("max slippage for %s is: %f\n", marketp.GetPair(), maxSlippage)

	lo := big.NewInt(1)
	hi := new(big.Int).Set(amountIn)
	var best lqtypes.PoolEdge
	found := false

	for i := 0; i < 12 && lo.Cmp(hi) <= 0; i++ {
		mid := new(big.Int).Rsh(new(big.Int).Add(lo, hi), 1)

		result, ok := AeroQuote(conn, marketp, quoterAddr, mid, oraclePrice, rateLimit)
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
	best.DexName = "AERODROME"
	return best, true
}

func AeroQuote(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	rateLimit time.Duration,
) (lqtypes.PoolEdge, bool) {
	var best lqtypes.PoolEdge
	found := false
	m := marketp

	for _, fee := range AerodromeFees {
		time.Sleep(rateLimit)
		result, ok := aeroQuoteCall(conn, m, quoterAddr, amountIn, oraclePrice, fee)
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
	best.AmountInOffset = 164 // même offset que Uniswap V3
	return best, found
}

func aeroQuoteCall(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	fee uint32,
) (lqtypes.PoolEdge, bool) {
	params := lqtypes.QuoteExactInputSingleParams{
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
