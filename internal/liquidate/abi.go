package liquidate

import (
	"log"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const liquidateABIJson = `[
  {
    "type": "function",
    "name": "liquidate",
    "inputs": [
      {
        "name": "marketParams",
        "type": "tuple",
        "components": [
          { "name": "loanToken", "type": "address" },
          { "name": "collateralToken", "type": "address" },
          { "name": "oracle", "type": "address" },
          { "name": "irm", "type": "address" },
          { "name": "lltv", "type": "uint256" }
        ]
      },
      {
        "name": "borrower",
        "type": "address"
      },
      {
        "name": "seizedAssets",
        "type": "uint256"
      },
      {
        "name": "repaidShares",
        "type": "uint256"
      },
      {
        "name": "steps",
        "type": "tuple[]",
        "components": [
          { "name": "target", "type": "address" },
          { "name": "data", "type": "bytes" },
          { "name": "tokenIn", "type": "address" },
          { "name": "tokenOut", "type": "address" },
          { "name": "amountInOffset", "type": "uint256" }
        ]
      },
      {
        "name": "minOut",
        "type": "uint256"
      }
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  }
]`

func LiquidatorAbiMethod() *abi.Method {
	LiquidatorAbi, err := abi.JSON(strings.NewReader(liquidateABIJson))
	if err != nil {
		log.Panic(err)
	}
	method := LiquidatorAbi.Methods["liquidate"]
	return &method
}
