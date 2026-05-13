package runner

import (
	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/internal/swap"
)

// swap  in:token address out:token address slippage
// double swap slippage_final = s1 + s2 - s1 * s2
/*
func (r *Runner) FindSwapRoutes() {

	singleHop := r.SingleHop()
	graph := swap.NewPoolGraph()

	for _, v := range singleHop {
		graph.AddPool(v)
	}
	for _, id := range r.Store.MarketReader.Ids() {
		morphoM := r.Store.MarketMap[id]
		k := lqtypes.RouteKey{
			TokenIn:  morphoM.CollateralToken,
			TokenOut: morphoM.LoanToken,
		}
		routes := graph.FindRoutes(morphoM.CollateralToken, morphoM.LoanToken, 1)
		r.SwapRoutes.Routes[k] = routes

	}
}
*/

func (r *Runner) SingleHop() {

	for _, id := range r.Store.MarketReader.Ids() {
		arr := make([]lqtypes.PoolEdge, 1)
		snap := r.Store.MarketReader.GetSnapshot(id)
		if snap == nil {
			continue
		}
		morphoM := r.Store.MarketMap[id]
		result, found := swap.QuoteBinarySearch(r.Infra.Conn, morphoM, r.Infra.Config.Addresses.UniSwapQuoter, snap.Stats.MaxCollateralPos, snap.Oracle.Price)
		if found {
			arr = append(arr, result)
		} else {
			r.Store.MarketReader.Update(morphoM.ID, func(m *cache.Market) {
				m.Canceled = true
			})
		}
		r.SwapRoutes.StoreRoute(arr)
	}

}
