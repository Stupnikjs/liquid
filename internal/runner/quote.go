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
// conn et cache
func (q *QuoteConsumer) QuotePools() {
	for id, m := range q.Cache.MarketMap {
		snap := q.Cache.Markets.GetSnapshot(id)
		log.Printf("quoting %s \n", m.GetPair())
		if snap == nil {
			continue
		}
		// add router address to struct
		for _, d := range q.Config.Dexs {
			result, found := d.BestAmountIn(q.Conn, &m, snap.Stats.MaxCollateralPos, snap.Oracle.Price, QUOTE_RATE_LIMIT)
			if found {
				result.DexName = d.DEX()
				q.Routes.AppendPool(result)
			}
		}
	}
	// quote bridge tokens

	err := utils.SavePoolEdgesJSON(q.Routes.Pools, "pools.json")
	if err != nil {
		log.Println("Err saving pools")
	}
	q.Routes.PoolsToGraph()
	// r.MarketConsumer SELECT MARKET WITH ROUTE
}

// cache
func (q *QuoteConsumer) SelectMarketWithRoute() {
	for _, m := range q.Cache.MarketMap {
		_, found := q.Routes.FindBestRoute(m.CollateralToken, m.LoanToken)
		log.Printf("checking route for %s found %v \n", m.GetPair(), found)
		if !found {
			q.Cache.Markets.Update(m.ID, func(m *cache.Market) {
				m.Canceled = true
			})
		}
	}
}

// for now we stay on single hop by parameters
