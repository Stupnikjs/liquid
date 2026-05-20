package liquidate

import (
	"fmt"
	"math/big"

	"github.com/Stupnikjs/liquid/internal/config/abi"
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

func BuildSteps(route swap.Route, liquidatorAddress common.Address) ([]SwapStep, error) {
	steps := make([]SwapStep, len(route.Hops))

	for i, hop := range route.Hops {
		// Encoder le calldata du DEX avec amountIn = 0 (placeholder)
		switch hop.DexName {
		case "UNIV3":
			data, err := abi.ExactInputSingle.EncodeArgs(abi.ExactInputSingleParams{
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
		case "AERO":

		case "SUSHI":

		default:
			return nil, fmt.Errorf("DEX non supporté: %s", hop.DexName)
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
