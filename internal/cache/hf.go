package cache

import (
	"math/big"
	"sort"

	"github.com/Stupnikjs/liquid/pkg/morpho"
)

// prec 1e18
func (m *Market) HF(pos *BorrowPosition) *big.Int {
    return morpho.HF(pos.CollateralAssets, pos.BorrowShares, m.Stats.TotalBorrowShares, m.Stats.TotalBorrowAssets, m.LLTV, m.Oracle.Price)
}

// recompute n hf from start
func (m *Market) RecomputeHFUnsafe(n int) {
	for i, p := range m.Positions {
		if i == n {
			break
		}
		hf := m.HF(p)
		p.CachedHF = hf // nil for non-borrowers, real value otherwise
	}
}

func (m *Market) SortAllPositionsByHFUnsafe() {

	sort.Slice(m.Positions, func(i, j int) bool {
		pi := m.Positions[i].CachedHF
		pj := m.Positions[j].CachedHF
		// nil traité comme zéro → rejeté en fin
		if pi == nil && pj == nil {
			return false
		}
		if pi == nil {
			return false
		}
		if pj == nil {
			return true
		}
		return pi.Cmp(pj) < 0
	})

}

// assume (risky) positions is sorted
func (s *MarketSnapshot) GetFirstHF() *big.Int {
	for _, p := range s.Positions {
		if p.CachedHF != nil {
			return p.CachedHF
		}
	}
	return nil
}
