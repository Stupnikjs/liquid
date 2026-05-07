package swap

import "github.com/ethereum/go-ethereum/common"

// swap  in:token address out:token address slippage
// double swap slippage_final = s1 + s2 - s1 * s2

type SwapAction struct {
	Router common.Address
}

type SwapResult struct {
	route [3]common.Address
	swaps []SwapAction
}

// Struct with all token
