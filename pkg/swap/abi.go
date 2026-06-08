package swap

import (
	"math/big"
	"strings"

	"github.com/Stupnikjs/liquid/pkg/ether"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type OnChainResult struct {
	AmountOut               *big.Int
	SqrtPriceX96After       *big.Int
	InitializedTicksCrossed uint32
	GasEstimate             *big.Int
}

func newResult() *OnChainResult {
	return &OnChainResult{
		AmountOut:               new(big.Int),
		SqrtPriceX96After:       new(big.Int),
		InitializedTicksCrossed: 0,
		GasEstimate:             new(big.Int),
	}
}

func QuoteExactInputSingleCall(
	quoter common.Address,
	params QuoteExactInputSingleParams,
	res *OnChainResult,
	quoteMethod *abi.Method,
) (*ether.AbiCall, error) {

	return ether.NewABICall(
		quoter,
		quoteMethod,
		[]any{params},
		func(outputs abi.Arguments, data []byte) error {
			values, err := outputs.Unpack(data)
			if err != nil {
				return err
			}
			res.AmountOut = values[0].(*big.Int)
			res.SqrtPriceX96After = values[1].(*big.Int)
			res.InitializedTicksCrossed = values[2].(uint32)
			res.GasEstimate = values[3].(*big.Int)
			return nil
		},
	)
}

func loadQuoteExactInputSingleMethod() (abi.Method, error) {
	const abiJSON = `[{
        "name": "quoteExactInputSingle",
        "type": "function",
        "inputs": [{
            "name": "params",
            "type": "tuple",
            "components": [
                {"name": "tokenIn",           "type": "address"},
                {"name": "tokenOut",          "type": "address"},
                {"name": "amountIn",          "type": "uint256"},
                {"name": "fee",               "type": "uint24"},
                {"name": "sqrtPriceLimitX96", "type": "uint160"}
            ]
        }],
        "outputs": [
            {"name": "amountOut",              "type": "uint256"},
            {"name": "sqrtPriceX96After",      "type": "uint160"},
            {"name": "initializedTicksCrossed","type": "uint32"},
            {"name": "gasEstimate",            "type": "uint256"}
        ]
    }]`

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return abi.Method{}, err
	}
	return parsedABI.Methods["quoteExactInputSingle"], nil
}
