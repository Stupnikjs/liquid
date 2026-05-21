package runner

import (
	"log"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/utils"
)

const QUOTE_RATE_LIMIT = 400 * time.Millisecond

// quote to Pools array
// then append to graph
func (r *Runner) singleHop() {
	for id, m := range r.Cache.MarketMap {
		snap := r.Cache.Markets.GetSnapshot(id)
		if snap == nil {
			continue
		}
		// add router address to struct
		for _, d := range r.Infra.Config.Dexs {
			result, found := d.Quote(r.Infra.Conn, &m, snap.Stats.MaxCollateralPos, snap.Oracle.Price, QUOTE_RATE_LIMIT)
			if found {
				result.DexName = d.Name
				r.SwapRoutes.AppendPool(result)
			}
		}
	}
	// quote bridge tokens

	err := utils.SavePoolEdgesJSON(r.SwapRoutes.Pools, "pools.json")
	if err != nil {
		log.Println("Err saving pools")
	}
	r.SwapRoutes.PoolsToGraph()
}

// for now we stay on single hop by parameters
func (r *Runner) RouteCacheRefresh() {
	r.singleHop()
	for _, m := range r.Cache.MarketMap {
		routes := r.SwapRoutes.Graph.FindRoutes(m.CollateralToken, m.LoanToken, 1)

		// select best route
		if len(routes) == 0 {
			r.Cache.Markets.Update(m.ID, func(m *cache.Market) {
				m.Canceled = true
			})
			continue
		}
		r.SwapRoutes.StoreRoute(routes[0])
	}

}
