package core

import (
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

type Cache struct {
	Markets   *market.MarketStore
	marketMap map[[32]byte]morpho.MarketParams
}

func NewCache(markets []morpho.MarketParams, config morpho.ChainConfig) *Cache {
	marketMap := make(map[[32]byte]morpho.MarketParams, len(markets))
	for _, m := range markets {
		marketMap[m.ID] = m
	}
	return &Cache{
		Markets:   market.NewStore(markets),
		marketMap: marketMap, // immutable
	}
}

func (c *Cache) GetMorphoMarketFromId(id [32]byte) morpho.MarketParams {
	return c.marketMap[id]
}
