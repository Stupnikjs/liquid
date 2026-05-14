package lqtypes

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3/w3types"
)

type EthCaller interface {
	EthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error
	FallBackEthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error
	SubscribeToEventPos(ctx context.Context, conf Config)
}

type MorphoMarket interface {
	GetPair() string
	GetCollateralToken() common.Address
	GetLoanToken() common.Address
	GetLLTV() *big.Int
}
