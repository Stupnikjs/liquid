package lqtypes

import (
	"context"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/lmittmann/w3/w3types"
)

type EthCaller interface {
	EthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error
	FallBackEthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error
}

type MarketReader interface {
	Ids() [][32]byte
	GetSnapshot(id [32]byte) *cache.MarketSnapshot
	Update(id [32]byte, fn func(m *cache.Market))
}
