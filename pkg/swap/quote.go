package swap

import (
	"math/big"
)

// ---------------------------------------------------------------------------
// Slippage
// ---------------------------------------------------------------------------

// ComputeSlippage returns the slippage percentage between the oracle-expected
// output and the actual amountOut returned by the DEX.
// oraclePrice is expected to be scaled at 1e36 / decimalsCollateral * decimalsLoan
// as returned by MorphoChainlinkOracleV2.
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
