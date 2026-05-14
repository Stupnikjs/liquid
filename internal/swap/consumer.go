package swap

import (
	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
)

type Consumer struct {
	Infra      *lqtypes.Infra
	Cache      *cache.Cache
	RouteCache *RouteCache
	Logger     chan string
}

func NewConsumer(infra *lqtypes.Infra, cache *cache.Cache, swapCache *RouteCache, logger chan string) *Consumer {
	return &Consumer{
		Infra:      infra,
		Logger:     logger,
		RouteCache: swapCache,
		Cache:      cache,
	}
}

func (c *Consumer) singleHop() {
	for id, m := range c.Cache.MarketMap {
		snap := c.Cache.Markets.GetSnapshot(id)
		if snap == nil {
			continue
		}
		// add router address to struct
		for _, q := range c.Infra.Config.Quoters {
			result, found := q.QuoterFunc(c.Infra.Conn, &m, q.QuoterAddr, snap.Stats.MaxCollateralPos, snap.Oracle.Price)
			if found {
				c.RouteCache.AppendPool(result)
			}

		}
	}
	c.RouteCache.PoolsToGraph()
}

func (c *Consumer) RouteCacheRefresh() {
	c.singleHop()
	for _, m := range c.Cache.MarketMap {
		routes := c.RouteCache.Graph.FindRoutes(m.CollateralToken, m.LoanToken, 4)
		// select best route
		if len(routes) == 0 {
			c.Cache.Markets.Update(m.ID, func(m *cache.Market) {
				m.Canceled = true
			})
			continue
		}
		c.RouteCache.StoreRoute(routes[0])
	}

}
