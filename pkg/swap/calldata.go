package swap

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	RouterUniswap   = uint8(0)
	RouterAerodrome = uint8(1)
)

func EncodeSwapData(tokenIn, tokenOut common.Address, routerType uint8, path []byte) ([]byte, error) {
	addressType, _ := abi.NewType("address", "", nil)
	uint8Type, _ := abi.NewType("uint8", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)

	args := abi.Arguments{
		{Type: addressType},
		{Type: addressType},
		{Type: uint8Type},
		{Type: bytesType},
	}

	return args.Pack(tokenIn, tokenOut, routerType, path)
}
