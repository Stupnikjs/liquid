package ether

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type CallMsg struct {
	To   common.Address `json:"to"`
	Data hexutil.Bytes  `json:"data"`
}

type AbiCall struct {
	Elem   rpc.BatchElem
	Raw    *hexutil.Bytes          // pointe sur Result dans elem
	Decode func(data []byte) error // spécifique à chaque call
}

func (a *AbiCall) Run() error {
	if a.Elem.Error != nil {
		return fmt.Errorf("rpc error: %w", a.Elem.Error)
	}
	return a.Decode([]byte(*a.Raw))
}

func NewABICall(
	to common.Address,
	method *abi.Method,
	args []any,
	decode func(outputs abi.Arguments, data []byte) error,
) (*AbiCall, error) {
	input, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, fmt.Errorf("pack %s: %w", method.Name, err)
	}
	calldata := append(method.ID, input...)

	raw := new(hexutil.Bytes)
	return &AbiCall{
		Elem: rpc.BatchElem{
			Method: "eth_call",
			Args:   []interface{}{CallMsg{To: to, Data: calldata}, "latest"},
			Result: raw,
		},
		Raw: raw,
		Decode: func(data []byte) error {
			return decode(method.Outputs, data)
		},
	}, nil
}
