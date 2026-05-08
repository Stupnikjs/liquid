pacakge swap 


type Consumer struct {
	Conn *connector.Connector 
	Config config.Config 
    Cache cache.MarketReader
	Logger chan string 
}


func (c *Consumer) Run () {
	manager := swap.NewSwapManager()
	// quoteSingle over markets 
	result, found := swap.QuoteBinarySearch(r.Conn, morphoM, r.Config.Addresses.UniSwapQuoter, snap.Stats.MaxCollateralPos, snap.Oracle.Price)
    // manager.Graph.AddPool(pool)
	for _, id := range r.Cache.Markets.Ids() {
		morphoM := r.Cache.GetMorphoMarketFromId(id)
		route := manager.Graph.FindRoutes(morphoM.CollateralToken, morphoM.LoanToken, 4)

	}

	// liquidgraph
	snap := r.Cache.Markets.GetSnapshot(id)
	if snap == nil {
		return
	}
	morphoM := r.Cache.MarketMap[id]
	
	if !found {
		r.Cache.Markets.Update(id, func(m *cache.Market) {
			m.Canceled = true
		})
		return
	}
	r.Cache.Markets.Update(id, func(m *cache.Market) {
		m.Stats.MaxUniSwappable = result.WCAmountIn
		m.Stats.SwapFee = result.Fee
	})

}