package runner

import (
	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

type Cache struct {
	Markets   *market.MarketStore
	marketMap map[[32]byte]morpho.MarketParams
}

func NewCache(conn *connector.Connector, conf config.Config, filters api.MarketFilters) *Cache {
	result, err := api.QueryMarkets(conn.ClientHTTP, conf.ChainID)
	if err != nil {
		return nil
	}
	markets := api.FilterMarket(result, filters, conf.ChainID)

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
