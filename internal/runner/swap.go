package runner

import (
	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/internal/swap"
)

// swap  in:token address out:token address slippage
// double swap slippage_final = s1 + s2 - s1 * s2

func (r *Runner) FindSwapRoutes() {

	swapMap := r.SingleHop()
	graph := swap.NewPoolGraph()
	for _, v := range swapMap {
		graph.AddPool(v)
	}
	for _, id := range r.Cache.Markets.Ids() {
		morphoM := r.Cache.GetMorphoMarketFromId(id)
		routes := graph.FindRoutes(morphoM.CollateralToken, morphoM.LoanToken, 1)

		r.Cache.Markets.Update(id, func(m *cache.Market) {
			if len(routes) == 0 {
				m.Canceled = true
				return
			}
			// need to put routes somewhere
		})
	}
}

func (r *Runner) SingleHop() []lqtypes.PoolEdge {
	arr := make([]lqtypes.PoolEdge, len(r.Store.MarketReader.Ids()))
	for _, id := range r.Store.MarketReader.Ids() {
		snap := r.Store.MarketReader.GetSnapshot(id)
		if snap == nil {
			continue
		}
		morphoM := r.Store.MarketMap[id]
		result, found := swap.QuoteBinarySearch(r.Conn, morphoM, r.Config.Addresses.UniSwapQuoter, snap.Stats.MaxCollateralPos, snap.Oracle.Price)
		if found {
			arr = append(arr, result)
		}

	}
	return arr

}
