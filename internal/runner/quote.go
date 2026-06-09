package runner

import (
	"log"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
)

const QUOTE_RATE_LIMIT = 700 * time.Millisecond

func (q *QuoteConsumer) QuotePools() {
	for id, m := range q.Cache.MarketMap {
		snap := q.Cache.Markets.GetSnapshot(id)
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

	q.Routes.PoolsToGraph()
	q.SelectMarketWithRoute()
}

// cache
func (q *QuoteConsumer) SelectMarketWithRoute() {
	for _, m := range q.Cache.MarketMap {
		route, found := q.Routes.FindBestRoute(m.CollateralToken, m.LoanToken)
		if found {
			log.Printf("checking route for %s route of len %d \n", m.GetPair(), len(route))
		}
		if !found {
			q.Cache.Markets.Update(m.ID, func(m *cache.Market) {
				m.Canceled = true
			})
		}
	}
}

// for now we stay on single hop by parameters
