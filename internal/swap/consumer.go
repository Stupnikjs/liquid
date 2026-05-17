package swap

import (
	"fmt"

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

// quote to Pools array
// then append to graph
func (c *Consumer) singleHop() {
	for id, m := range c.Cache.MarketMap {
		snap := c.Cache.Markets.GetSnapshot(id)
		if snap == nil {
			continue
		}
		// add router address to struct
		for _, d := range c.Infra.Config.Dexs {
			result, found := d.Quote(c.Infra.Conn, m, d.QuoterAddress, snap.Stats.MaxCollateralPos, snap.Oracle.Price)
			if found {
				c.RouteCache.AppendPool(result)
			}

		}
	}
	c.RouteCache.PoolsToGraph()
}

// for now we stay on single hop by parameters
func (c *Consumer) RouteCacheRefresh() {
	c.singleHop()
	for _, m := range c.Cache.MarketMap {
		routes := c.RouteCache.Graph.FindRoutes(m.CollateralToken, m.LoanToken, 1)
		fmt.Println(routes)
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
