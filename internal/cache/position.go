package cache

import (
	"math/big"
	"sort"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
)

type BorrowPosition struct {
	MarketID                                 [32]byte
	Address                                  common.Address
	BorrowShares, CollateralAssets, CachedHF *big.Int
}

func (m *Market) InsertPositionUnsafe(pos *BorrowPosition) {
	index := len(m.Positions)

	if pos.CachedHF != nil {
		for i, p := range m.Positions {
			if p.CachedHF == nil || p.CachedHF.Cmp(pos.CachedHF) > 0 {
				index = i
				break
			}
		}
	}
	m.Positions = append(m.Positions, nil)
	copy(m.Positions[index+1:], m.Positions[index:])
	m.Positions[index] = pos
}

func (m *Market) InsertPosition(pos *BorrowPosition) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.InsertPositionUnsafe(pos)
}

func (m *Market) RemovePosition(addr common.Address) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	for i, p := range m.Positions {
		if p.Address == addr {
			m.Positions = append(m.Positions[:i], m.Positions[i+1:]...)
			break
		}
	}
}

func (m *Market) UpdatePosition(pos *BorrowPosition) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	for i, p := range m.Positions {
		if p.Address == pos.Address {
			m.Positions[i] = pos
			break
		}
	}
}

func (m *Market) GetBorrowPosition(addr common.Address) *BorrowPosition {
	m.Mu.RLock()
	defer m.Mu.RUnlock()
	for _, p := range m.Positions {
		if p.Address == addr {
			return p
		}
	}
	return nil
}

// prec 1e18
func (pos *BorrowPosition) HF(
	totShares, totBorrowAssets, oraclePrice, LLTV *big.Int,
) *big.Int {
	borrowAssets := morpho.BorrowAssetsFromShares(
		pos.BorrowShares, totShares, totBorrowAssets,
	)
	if borrowAssets == nil || pos.CollateralAssets == nil {
		return big.NewInt(0)
	}
	if borrowAssets.Sign() == 0 || pos.CollateralAssets.Sign() == 0 {
		return big.NewInt(0)
	}
	// numerator = collateral * price * LLTV
	numerator := new(big.Int).Mul(pos.CollateralAssets, oraclePrice)
	numerator.Mul(numerator, LLTV)
	// denominator = borrow * 1e36
	denominator := new(big.Int).Mul(borrowAssets, utils.TenPowInt(36))
	hf := new(big.Int).Div(numerator, denominator)
	// utils.BigIntToFloat(hf)/1e18
	return hf
}

func (m *Market) RecomputeAllHFUnsafe() {
	for _, p := range m.Positions {
		p.CachedHF = p.HF(m.Stats.TotalBorrowShares, m.Stats.TotalBorrowAssets, m.Oracle.Price, m.LLTV)
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

func (m *Market) SortAllPositionsByHF() {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.SortAllPositionsByHFUnsafe()
}
