package runner

import (
	"log"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/utils"
)

const QUOTE_RATE_LIMIT = 50 * time.Millisecond

// quote to Pools array
// then append to graph
func (r *Runner) QuotePools() {
	for id, m := range r.Cache.MarketMap {
		snap := r.Cache.Markets.GetSnapshot(id)
		log.Printf("quoting %s \n", m.GetPair())
		if snap == nil {
			continue
		}
		// add router address to struct
		for _, d := range r.Infra.Config.Dexs {
			result, found := d.BestAmountIn(r.Infra.Conn, &m, snap.Stats.MaxCollateralPos, snap.Oracle.Price, QUOTE_RATE_LIMIT)
			if found {
				result.DexName = d.DEX()
				r.Routes.AppendPool(result)
			}
		}
	}
	// quote bridge tokens

	err := utils.SavePoolEdgesJSON(r.Routes.Pools, "pools.json")
	if err != nil {
		log.Println("Err saving pools")
	}
	r.Routes.PoolsToGraph()
	r.SelectMarketWithRoute()
}

func (r *Runner) SelectMarketWithRoute() {
	for _, m := range r.Cache.MarketMap {
		_, found := r.Routes.FindBestRoute(m.CollateralToken, m.LoanToken)
		log.Printf("checking route for %s found %v \n", m.GetPair(), found)
		if !found {
			r.Cache.Markets.Update(m.ID, func(m *cache.Market) {
				m.Canceled = true
			})
		}
	}
}

// for now we stay on single hop by parameters
