package swap

import (
	"context"
	"fmt"
	"math/big"

	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

type QuoteExactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	AmountIn          *big.Int
	Fee               *big.Int
	SqrtPriceLimitX96 *big.Int
}

type QuoteResult struct {
	AmountIn     *big.Int
	AmountOut    *big.Int
	NetAmountOut *big.Int // après fees
	Fee          uint32
	FeeAmount    *big.Int // montant des fees en tokenIn
	Slippage     float64  // en %
}

var (
	UniswapQuoterV2Addr = common.HexToAddress("0x3d4e44Eb1374240CE5F1B871ab261CD16335B76a")

	FuncQuoteExactInputSingleV2 = w3.MustNewFunc(
		`quoteExactInputSingle((address tokenIn, address tokenOut, uint256 amountIn, uint24 fee, uint160 sqrtPriceLimitX96) params)`,
		`uint256 amountOut, uint160 sqrtPriceX96After, uint32 initializedTicksCrossed, uint256 gasEstimate`,
	)
)

var UniswapFees = []uint32{100, 500, 3000, 10000}

func MaxSlippage(lltv *big.Int) float64 {
	// lltv est en 1e18, ex: 945000000000000000 = 94.5%
	lltvF, _ := new(big.Float).SetInt(lltv).Float64()
	lltvPct := lltvF / 1e18 * 100 // ex: 94.5
	bonus := 100 - lltvPct        // ex: 5.5%
	gas := 0.1                    // ~0.1% pour le gas
	return bonus - gas            // ex: 5.4% max
}

func Quote(client *w3.Client, marketp morpho.MarketParams, amountIn, oraclePrice *big.Int) (*QuoteResult, error) {
	var best *QuoteResult
	current := new(big.Int).Set(amountIn)
	maxSlippage := MaxSlippage(marketp.LLTV)
	for current.Sign() > 0 {
		for _, fee := range UniswapFees {
			result, err := quoteSingle(client, marketp, current, oraclePrice, fee)
			if err != nil || result == nil {
				continue
			}
			if result.Slippage <= maxSlippage {
				fmt.Printf("Pair %s/%s | source: uniswap | slippage: %f%%\n",
					marketp.CollateralTokenStr, marketp.LoanTokenStr, result.Slippage)
				if best == nil || result.AmountOut.Cmp(best.AmountOut) > 0 {
					best = result
				}
			}
		}

		if best != nil {
			return best, nil
		}

		// divise par 2 et réessaie
		current.Div(current, big.NewInt(2))
		fmt.Printf("slippage trop élevé, on réessaie avec amountIn: %s\n", current.String())
	}

	return nil, fmt.Errorf("no acceptable slippage found for %s -> %s",
		marketp.CollateralTokenStr, marketp.LoanTokenStr)
}

func quoteSingle(client *w3.Client, marketp morpho.MarketParams, amountIn, oraclePrice *big.Int, fee uint32) (*QuoteResult, error) {
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

	if err := client.CallCtx(context.Background(),
		eth.CallFunc(UniswapQuoterV2Addr, FuncQuoteExactInputSingleV2, params).Returns(
			&amountOut,
			&sqrtPriceX96After,
			&initializedTicksCrossed,
			&gasEstimate,
		),
	); err != nil {
		return nil, nil // pool inexistante, on skip
	}

	// fees = amountIn * fee / 1_000_000
	feeAmount := new(big.Int).Mul(amountIn, big.NewInt(int64(fee)))
	feeAmount.Div(feeAmount, big.NewInt(1_000_000))

	// slippage via sqrtPriceX96After
	slippage := computeSlippage(amountIn, amountOut, oraclePrice)

	return &QuoteResult{
		AmountIn:     amountIn,
		AmountOut:    amountOut,
		NetAmountOut: amountOut, // déjà net de fees Uniswap
		Fee:          fee,
		FeeAmount:    feeAmount,
		Slippage:     slippage,
	}, nil
}

func computeSlippage(amountIn, amountOut *big.Int, oraclePrice *big.Int) float64 {
	if oraclePrice == nil || oraclePrice.Sign() == 0 {
		return 0
	}

	// expectedOut = amountIn * oraclePrice / 1e36
	expectedOut := new(big.Int).Mul(amountIn, oraclePrice)
	expectedOut.Div(expectedOut, new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil))

	if expectedOut.Sign() == 0 {
		return 0
	}

	// slippage = (expectedOut - amountOut) / expectedOut * 100
	diff := new(big.Int).Sub(expectedOut, amountOut)
	diffF := new(big.Float).SetInt(diff)
	expectedF := new(big.Float).SetInt(expectedOut)
	slippageF := new(big.Float).Quo(diffF, expectedF)
	slip, _ := slippageF.Float64()

	return slip * 100
}
