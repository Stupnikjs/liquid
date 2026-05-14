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
		for _, fn := range c.Infra.Config.Quoters {
			result, found := fn(c.Infra.Conn, m, c.Infra.Config.Addresses.UniSwapQuoter, snap.Stats.MaxCollateralPos, snap.Oracle.Price)
			if found {
				c.RouteCache.AppendPool(result)
			}

		}
	}
}

func (c *Consumer) RouteCacheUpdate() {
	c.singleHop()

}
