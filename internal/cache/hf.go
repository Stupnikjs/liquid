package cache

import (
	"math/big"
	"sort"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

// prec 1e18
func (m *Market) HF(pos *BorrowPosition) *big.Int {

	borrowAssets := morpho.BorrowAssetsFromShares(
		pos.BorrowShares, m.Stats.TotalBorrowShares, m.Stats.TotalBorrowAssets,
	)
	if borrowAssets == nil || pos.CollateralAssets == nil {
		return nil // ← nil, not 0
	}
	if borrowAssets.Sign() == 0 || pos.CollateralAssets.Sign() == 0 {
		return nil // ← nil, not 0
	}
	// numerator = collateral * price * LLTV
	numerator := new(big.Int).Mul(pos.CollateralAssets, m.Oracle.Price)
	numerator.Mul(numerator, m.LLTV)
	// denominator = borrow * 1e36
	denominator := new(big.Int).Mul(borrowAssets, utils.TenPowInt(36))
	hf := new(big.Int).Div(numerator, denominator)
	// utils.BigIntToFloat(hf)/1e18
	return hf
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

func (s *MarketSnapshot) GetFirstHF() *big.Int {
	for _, p := range s.Positions {
		if p.CachedHF != nil {
			return p.CachedHF // first non-nil HF (should be lowest after sort)
		}
	}
	return nil
}
