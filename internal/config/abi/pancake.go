package abi

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

var PancakeExactInputSingle = w3.MustNewFunc(`exactInputSingle((
    address tokenIn,
    address tokenOut,
    uint24  fee,
    address recipient,
    uint256 deadline,
    uint256 amountIn,
    uint256 amountOutMinimum,
    uint160 sqrtPriceLimitX96
))`, "uint256")

type PancakeExactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	Fee               *big.Int
	Recipient         common.Address
	Deadline          *big.Int
	AmountIn          *big.Int
	AmountOutMinimum  *big.Int
	SqrtPriceLimitX96 *big.Int
}
