package market

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
)

func NewStore(markets []morpho.MarketParams) *MarketStore {
	marketsMap := make(map[[32]byte]*Market, len(markets))
	for _, m := range markets {
		market := &Market{
			Positions: make(map[common.Address]*BorrowPosition),
		}
		marketsMap[m.ID] = market
	}

	return &MarketStore{
		mu:      sync.RWMutex{},
		markets: marketsMap,
	}
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
		},
		Positions: make([]BorrowPosition, 0, len(market.Positions)),
	}

	for _, p := range market.Positions {

		snap.Positions = append(snap.Positions, *p)
	}

	return snap
}

func (s *MarketStore) GetPositions(id [32]byte) []BorrowPosition {
	s.mu.RLock()
	market := s.markets[id]
	s.mu.RUnlock()

	if market == nil {
		return nil
	}

	market.Mu.RLock()
	defer market.Mu.RUnlock()

	if market.Canceled ||
		market.LLTV == nil ||
		market.Stats.TotalBorrowAssets == nil ||
		market.Stats.TotalBorrowShares == nil {
		return nil
	}

	Positions := make([]BorrowPosition, 0, len(market.Positions))

	for _, p := range market.Positions {
		Positions = append(Positions, *p)
	}

	return Positions
}

func (s *MarketStore) CleanNonSwap(id [32]byte) error {
	s.mu.RLock()
	market := s.markets[id]
	s.mu.RUnlock()

	if market == nil {
		return nil
	}

	// 1. collecter les adresses à supprimer sans lock d'écriture
	market.Mu.RLock()
	if market.Canceled ||
		market.LLTV == nil ||
		market.Stats.TotalBorrowAssets == nil ||
		market.Stats.TotalBorrowShares == nil ||
		market.Stats.MaxCollateralPos == nil ||
		market.Stats.MaxUniSwappable == nil {
		market.Mu.RUnlock()
		return fmt.Errorf("market stats uncomplete")
	}

	toDelete := make([]common.Address, 0)
	for addr, p := range market.Positions {
		if p.CollateralAssets.Cmp(market.Stats.MaxUniSwappable) > 0 {
			toDelete = append(toDelete, addr)
		}
	}
	market.Mu.RUnlock()

	// 2. supprimer en dehors de la boucle avec le bon lock
	if len(toDelete) > 0 {
		if len(toDelete) > 0 {
			s.Update(id, func(m *Market) {
				for _, addr := range toDelete {
					delete(m.Positions, addr)
				}
			})
		}

	}

	return nil
}
