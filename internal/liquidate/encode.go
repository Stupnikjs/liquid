package liquidate

import (
	"fmt"
	"math/big"

	"github.com/Stupnikjs/liquid/internal/config/abi"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/Stupnikjs/liquid/pkg/swap"
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

func (c *Consumer) ToLiquidationArg(l *Liquidable, params morpho.MarketParams, route []swap.PoolEdge) ([]byte, error) {
	m := params.ToMarketContractParams()
	steps, err := BuildSteps(route, c.Infra.Config.Addresses.LiquidatorContract)
	if err != nil {
		return nil, fmt.Errorf("build steps: %w", err)
	}
	return BuildLiquidateCalldata(*m, l.Pos.Address, l.SeizeAssets, l.RepayShares, steps, new(big.Int).SetInt64(0))
}

// converting route to contract params swap step
func BuildSteps(route []swap.PoolEdge, liquidatorAddress common.Address) ([]swap.SwapStep, error) {
	steps := make([]swap.SwapStep, len(route))

	for i, hop := range route {
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

			steps[i] = swap.SwapStep{
				Target:         hop.Router,
				Data:           data,
				TokenIn:        hop.TokenIn,
				TokenOut:       hop.TokenOut,
				AmountInOffset: big.NewInt(132), // 32 + 4 + 4*32 pour exactInputSingle
			}

		case "PANCAKE":
			data, err := abi.PancakeExactInputSingle.EncodeArgs(abi.PancakeExactInputSingleParams{
				TokenIn:           hop.TokenIn,
				TokenOut:          hop.TokenOut,
				Fee:               big.NewInt(int64(hop.Fee)),
				Recipient:         liquidatorAddress,
				Deadline:          big.NewInt(0), // placeholder, patché on-chain
				AmountIn:          big.NewInt(0), // placeholder, patché on-chain
				AmountOutMinimum:  big.NewInt(0),
				SqrtPriceLimitX96: big.NewInt(0),
			})
			if err != nil {
				return nil, err
			}

			steps[i] = swap.SwapStep{
				Target:         hop.Router,
				Data:           data,
				TokenIn:        hop.TokenIn,
				TokenOut:       hop.TokenOut,
				AmountInOffset: big.NewInt(164), // 32 + 4 + 4*32 pour exactInputSingle
			}

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
	steps []swap.SwapStep,
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
