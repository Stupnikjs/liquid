package runner

import (
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

type Cache struct {
	Markets   *market.MarketStore
	marketMap map[[32]byte]morpho.MarketParams
}

func NewCache(markets []morpho.MarketParams) *Cache {
	marketMap := make(map[[32]byte]morpho.MarketParams, len(markets))
	store := market.NewStore(markets)
	for _, mk := range markets {
		marketMap[mk.ID] = mk
		store.Update(mk.ID, func(m *market.Market) {
			m.LLTV = mk.LLTV
			m.Oracle.Address = mk.Oracle
		})
	}

	return &Cache{
		Markets:   store,
		marketMap: marketMap, // immutable
	}
}

func (c *Cache) GetMorphoMarketFromId(id [32]byte) morpho.MarketParams {
	return c.marketMap[id]
}
