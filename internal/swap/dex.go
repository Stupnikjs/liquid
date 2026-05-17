package swap

import (
	"context"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

type Dex struct {
	QuoterAddr common.Address
	RouterAddr common.Address
	Quoter     QuoterFunc
}

type QuoterFunc func(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
	maxSlippage float64,
	rateLimit time.Duration,
) (lqtypes.PoolEdge, bool)

type QuotSingleFunc func(conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	fee uint32,
) (lqtypes.PoolEdge, bool)

func (d *Dex) Quote(conn lqtypes.EthCaller, marketp lqtypes.MorphoMarket, amountIn, oraclePrice *big.Int, maxSlippage float64,
	rateLimit time.Duration) (lqtypes.PoolEdge, bool) {
	return d.Quoter(conn, marketp, d.QuoterAddr, amountIn, oraclePrice, maxSlippage, rateLimit)
}

func UniQuote(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	maxSlippage float64,
	rateLimit time.Duration,
) (lqtypes.PoolEdge, bool) {
	var best lqtypes.PoolEdge
	var UniswapFees = []uint32{100, 500, 3000, 10000}
	found := false

	for _, fee := range UniswapFees {
		time.Sleep(rateLimit)
		result, ok := uniQuoteCall(conn, marketp, quoterAddr, amountIn, oraclePrice, fee)
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
