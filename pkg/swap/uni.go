package swap

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

var (
	FuncQuoteExactInputSingle = w3.MustNewFunc(`quoteExactInputSingle((address tokenIn, address tokenOut, uint256 amountIn, uint24 fee, uint160 sqrtPriceLimitX96))`, `uint256 amountOut, uint160 sqrtPriceX96After, uint32 initializedTicksCrossed, uint256 gasEstimate`)
	QuoterV2Addr              = common.HexToAddress("0x3d4e44Eb1374240CE5F1B871ab261CD16335B76a") // Base
)

type QuoteResult struct {
	AmountOut               *big.Int
	SqrtPriceX96After       *big.Int
	InitializedTicksCrossed uint32
	GasEstimate             *big.Int
}

func QuoteSwap(client *w3.Client, tokenIn, tokenOut common.Address, amountIn *big.Int, fee uint32) (*QuoteResult, error) {
	type QuoteParams struct {
		TokenIn           common.Address
		TokenOut          common.Address
		AmountIn          *big.Int
		Fee               *big.Int
		SqrtPriceLimitX96 *big.Int
	}

	params := QuoteParams{
		TokenIn:           tokenIn,
		TokenOut:          tokenOut,
		AmountIn:          amountIn,
		Fee:               big.NewInt(int64(fee)),
		SqrtPriceLimitX96: big.NewInt(0),
	}

	var amountOut *big.Int
	// ... autres return values

	if err := client.CallCtx(context.Background(),
		eth.CallFunc(QuoterV2Addr, FuncQuoteExactInputSingle, params).Returns(&amountOut),
	); err != nil {
		return nil, err
	}

	return &QuoteResult{AmountOut: amountOut}, nil
}

func FindBestPool(client *w3.Client, tokenIn, tokenOut common.Address, amountIn *big.Int, oraclePrice *big.Int) (bestFee uint32, bestSlippage float64) {

	fees := []uint32{100, 500, 3000, 10000} // 0.01%, 0.05%, 0.3%, 1%
	bestSlippage = 100.0

	// expectedOut = amountIn * oraclePrice / 1e36
	fmt.Println(oraclePrice)
	pow36 := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	expectedOut := new(big.Int).Mul(amountIn, oraclePrice)
	expectedOut.Div(expectedOut, pow36)

	if expectedOut.Sign() == 0 {
		fmt.Println("expected out 0")
		return 0, 100.0
	}

	for _, fee := range fees {
		result, err := QuoteSwap(client, tokenIn, tokenOut, amountIn, fee)
		if err != nil || result.AmountOut == nil || result.AmountOut.Sign() == 0 {
			continue
		}

		// slippage = (expected - actual) / expected * 100
		diff := new(big.Int).Sub(expectedOut, result.AmountOut)
		slippagePct := new(big.Float).Quo(
			new(big.Float).SetInt(diff),
			new(big.Float).SetInt(expectedOut),
		)
		fmt.Println(slippagePct.Float32())
		slippagePct.Mul(slippagePct, big.NewFloat(100))
		slip, _ := slippagePct.Float64()

		if slip < bestSlippage {
			bestSlippage = slip
			bestFee = fee
		}
	}
	return bestFee, bestSlippage
}
