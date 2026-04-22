package cache

import (
	"math/big"

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
	// copy first part after index , insert new pos at index, rest already copied after index+1
	copy(m.Positions[index+1:], m.Positions[index:])
	m.Positions[index] = pos
}

func (m *Market) InsertPosition(pos *BorrowPosition) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	pos.CachedHF = m.HF(pos)
	m.InsertPositionUnsafe(pos)
}

func (m *Market) RemovePositionUnsafe(addr common.Address) {
	for i, p := range m.Positions {
		if p.Address == addr {
			m.Positions = append(m.Positions[:i], m.Positions[i+1:]...)
			break
		}
	}
}

func (m *Market) RemovePosition(addr common.Address) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.RemovePositionUnsafe(addr)
}

func (m *Market) UpdatePositionUnsafe(pos *BorrowPosition) {
	m.RemovePositionUnsafe(pos.Address)
	pos.CachedHF = m.HF(pos)
	m.InsertPositionUnsafe(pos)
}

func (m *Market) UpdatePosition(pos *BorrowPosition) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.UpdatePositionUnsafe(pos)
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
