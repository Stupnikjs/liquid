package liquidate

import (
	"math/big"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/internal/swap"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

// ABI de la fonction liquidate
var liquidateFunc = w3.MustNewFunc(`liquidate(
    (address loanToken, address collateralToken, address oracle, address irm, uint256 lltv) marketParams,
    address borrower,
    uint256 seizedAssets,
    uint256 repaidShares,
    (address target, bytes data, address tokenIn, address tokenOut, uint256 amountInOffset)[] steps,
    uint256 minOut
)`, "")

// Struct Go qui mappe SwapStep
type SwapStep struct {
	Target         common.Address
	Data           []byte
	TokenIn        common.Address
	TokenOut       common.Address
	AmountInOffset *big.Int
}

// Encoder le calldata Uniswap du step
var exactInputSingle = w3.MustNewFunc(`exactInputSingle((
    address tokenIn,
    address tokenOut,
    uint24  fee,
    address recipient,
    uint256 amountIn,
    uint256 amountOutMinimum,
    uint160 sqrtPriceLimitX96
))`, "uint256")

type ExactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	Fee               *big.Int
	Recipient         common.Address
	AmountIn          *big.Int
	AmountOutMinimum  *big.Int
	SqrtPriceLimitX96 *big.Int
}

func BuildSteps(route swap.Route, liquidatorAddress common.Address) ([]SwapStep, error) {
	steps := make([]SwapStep, len(route.Hops))

	for i, hop := range route.Hops {
		// Encoder le calldata du DEX avec amountIn = 0 (placeholder)
		data, err := exactInputSingle.EncodeArgs(ExactInputSingleParams{
			TokenIn:           hop.TokenIn,
			TokenOut:          hop.TokenOut,
			Fee:               big.NewInt(int64(hop.Fee)),
			Recipient:         liquidatorAddress,
			AmountIn:          big.NewInt(0), // placeholder, patché on-chain
			AmountOutMinimum:  big.NewInt(0),
			SqrtPriceLimitX96: big.NewInt(0),
		})
		if err != nil {
			return nil, err
		}

		steps[i] = SwapStep{
			Target:         hop.Router,
			Data:           data,
			TokenIn:        hop.TokenIn,
			TokenOut:       hop.TokenOut,
			AmountInOffset: big.NewInt(hop.AmountInOffset), // 32 + 4 + 4*32 pour exactInputSingle
		}
	}

	return steps, nil
}

func BuildLiquidateCalldata(
	marketParams lqtypes.MarketContractParams,
	borrower common.Address,
	seizedAssets *big.Int,
	repaidShares *big.Int,
	steps []SwapStep,
	minOut *big.Int,
) ([]byte, error) {
	return liquidateFunc.EncodeArgs(
		marketParams,
		borrower,
		seizedAssets,
		repaidShares,
		steps,
		minOut,
	)
}
