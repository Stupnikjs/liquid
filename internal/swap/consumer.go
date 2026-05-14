package swap

import (
	"github.com/Stupnikjs/liquid/internal/lqtypes"
)

type Consumer struct {
	Infra      *lqtypes.Infra
	Store      lqtypes.Store
	RouteCache *RouteCache
	Logger     chan string
}

func NewConsumer(infra *lqtypes.Infra, store lqtypes.Store, swapCache *RouteCache, logger chan string) *Consumer {
	return &Consumer{
		Infra:      infra,
		Logger:     logger,
		RouteCache: swapCache,
		Store:      store,
	}
}

func (c *Consumer) singleHop() {
	for id, m := range c.Store.MarketMap {
		snap := c.Store.MarketReader.GetSnapshot(id)
		if snap == nil {
			continue
		}
		// add router address to struct
		for _, q := range c.Infra.Config.Quoters {
			result, found := q.Fn(c.Infra.Conn, m, q.QuoterAddr, snap.Stats.MaxCollateralPos, snap.Oracle.Price)
			if found {
				c.RouteCache.AppendPool(result)
			}

		}
	}
}

func (c *Consumer) RouteCacheUpdate() {
	c.singleHop()
	for _, m := range c.Store.MarketMap {
		routes := c.RouteCache.Graph.FindRoutes(m.CollateralToken, m.LoanToken, 4)
		// select best route
		c.RouteCache.StoreRoute(routes[0])
	}

}
