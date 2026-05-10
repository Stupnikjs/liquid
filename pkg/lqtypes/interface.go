package lqtypes

import (
	"context"

	"github.com/lmittmann/w3/w3types"
)

type EthCaller interface {
	EthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error
	FallBackEthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error
}
