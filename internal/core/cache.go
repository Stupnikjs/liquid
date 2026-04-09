package core

import (
	"context"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/w3types"
)

type Cache struct {
	Markets *market.MarketStore
}

func NewCache(markets []morpho.MarketParams, config morpho.ChainConfig) *Cache {
	return &Cache{
		Markets: market.NewStore(len(markets)),
	}
}

// func (c *w3.Client) CallCtx(ctx context.Context, calls ...w3types.RPCCaller) error
func (c *Cache) EthCallCtx(client *w3.Client, ctx context.Context, calls []w3types.RPCCaller) error {

	return client.CallCtx(ctx, calls...)
}
