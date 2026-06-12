package runner

import (
	"log"
	"time"
)

const QUOTE_RATE_LIMIT = 500 * time.Millisecond

func (q *QuoteConsumer) QuotePools() {
	for _, m := range q.Cache.AllMarkets() {
		snap := q.Cache.GetSnapshot(m.ID)
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
	for _, m := range q.Cache.AllMarkets() {
		route, found := q.Routes.FindBestRoute(m.CollateralToken, m.LoanToken)
		if found {
			log.Printf("checking route for %s route of len %d \n", m.GetPair(), len(route))
		}
		if !found {

			q.Cache.CancelMarket(m.ID)
		}
	}
}

// for now we stay on single hop by parameters
