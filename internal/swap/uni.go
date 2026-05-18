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

func UniQuoter(
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

		result, ok := UniQuote(conn, marketp, quoterAddr, mid, oraclePrice, rateLimit)
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

func UniQuote(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	rateLimit time.Duration,
) (lqtypes.PoolEdge, bool) {
	var best lqtypes.PoolEdge
	var UniswapFees = []uint32{100, 500, 3000, 10000}
	found := false
	m := marketp
	for _, fee := range UniswapFees {
		time.Sleep(rateLimit)
		result, ok := uniQuoteCall(conn, m, quoterAddr, amountIn, oraclePrice, fee)
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
