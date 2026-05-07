package cache

import (
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/morpho"
)

// changer map par array avec HF triée

func NewStore(markets []morpho.MarketParams) *MarketStore {
	marketsMap := make(map[[32]byte]*Market, len(markets))
	for _, m := range markets {

		market := &Market{
			// might inizialize array
			Positions: make([]*BorrowPosition, 0),
		}
		marketsMap[m.ID] = market
	}

	return &MarketStore{
		mu:      sync.RWMutex{},
		markets: marketsMap,
	}
}
func (s *MarketStore) AllPosLen() int {
	s.mu.RLock()
	sum := 0
	for _, m := range s.markets {
		sum += len(m.Positions)
	}
	s.mu.RUnlock()
	return sum

}

func (s *MarketStore) Range(fn func(id [32]byte)) {
	ids := s.Ids()
	for _, id := range ids {
		fn(id)
	}
}

func (s *MarketStore) Ids() [][32]byte {
	s.mu.RLock()
	ids := make([][32]byte, 0, len(s.markets))
	for id, m := range s.markets {
		if !m.Canceled {
			ids = append(ids, id)
		}
	}
	s.mu.RUnlock()
	return ids

}

func (s *MarketStore) Upsert(id [32]byte, m *Market) {
	s.mu.Lock()
	s.markets[id] = m
	s.mu.Unlock()
}

func (s *MarketStore) Update(id [32]byte, fn func(m *Market)) {
	s.mu.RLock()
	m := s.markets[id]
	s.mu.RUnlock()

	if m == nil {
		return
	}

	m.Mu.Lock()
	fn(m)
	m.Mu.Unlock()
}

func (s *MarketStore) GetSnapshot(id [32]byte) *MarketSnapshot {
	s.mu.RLock()
	market := s.markets[id]
	s.mu.RUnlock()

	if market == nil {
		return nil
	}

	market.Mu.RLock()
	defer market.Mu.RUnlock()
	if market.Canceled ||
		market.Oracle.Price == nil ||
		market.LLTV == nil ||
		market.Stats.TotalBorrowAssets == nil ||
		market.Stats.TotalBorrowShares == nil || market.Stats.MaxCollateralPos == nil {
		return nil
	}

	if market.Stats.MaxUniSwappable == nil {
		market.Stats.MaxUniSwappable = big.NewInt(0)

	}

	snap := &MarketSnapshot{
		ID: id,
		Oracle: Oracle{
			Price:   new(big.Int).Set(market.Oracle.Price),
			Address: market.Oracle.Address,
		},
		LLTV: new(big.Int).Set(market.LLTV),
		Stats: MarketStats{
			TotalBorrowAssets: new(big.Int).Set(market.Stats.TotalBorrowAssets),
			TotalBorrowShares: new(big.Int).Set(market.Stats.TotalBorrowShares),
			MaxCollateralPos:  new(big.Int).Set(market.Stats.MaxCollateralPos),
			MaxUniSwappable:   new(big.Int).Set(market.Stats.MaxUniSwappable),
			SwapFee:           market.Stats.SwapFee,
		},
		Positions: make([]BorrowPosition, 0, len(market.Positions)),
	}
	// anyway to avoid loop
	for _, p := range market.Positions {
		snap.Positions = append(snap.Positions, *p)
	}

	return snap
}

func GetMarketLog(snap MarketSnapshot, id [32]byte, morphoM morpho.MarketParams) string {

	var sb strings.Builder
	marketPair := fmt.Sprintf("%s/%s ", morphoM.CollateralTokenStr, morphoM.LoanTokenStr)

	fmt.Fprintf(&sb, "%s %d pos ", marketPair, len(snap.Positions))
	priceStr := "nil"
	if snap.Oracle.Price != nil {
		priceStr = utils.FormatDecimals(
			snap.Oracle.Price,
			int(36+morphoM.LoanTokenDecimals-morphoM.CollateralTokenDecimals),
		)
	}
	limit := min(len(snap.Positions), 1)
	for i := range limit {
		pos := snap.Positions[i]
		fmt.Fprintf(&sb,
			"BorrowShares=%s, Collateral=%s, Oracle=%s  , HF=%s\n",
			utils.FormatDecimals(pos.BorrowShares, 18),
			utils.FormatDecimals(pos.CollateralAssets, int(morphoM.CollateralTokenDecimals)),
			priceStr,
			utils.FormatDecimals(pos.CachedHF, 18),
		)
	}
	return sb.String()
}
