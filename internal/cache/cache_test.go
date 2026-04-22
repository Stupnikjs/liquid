package cache

import (
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func newMarketID(b byte) [32]byte {
	var id [32]byte
	id[0] = b
	return id
}

func newBorrowPosition(addr common.Address, hf int64, collateral int64) *BorrowPosition {
	return &BorrowPosition{
		Address:         addr,
		CachedHF:        big.NewInt(hf),
		CollateralAssets: big.NewInt(collateral),
	}
}

func newMarket() *Market {
	return &Market{
		Canceled:    false,
		Sorted:      make([]*BorrowPosition, 0),
		ActiveLimit: 0,
		Oracle: Oracle{
			Price:   big.NewInt(1e18),
			Address: common.Address{},
		},
		LLTV: big.NewInt(8e17), // 0.8
		Stats: MarketStats{
			TotalBorrowAssets: big.NewInt(1000),
			TotalBorrowShares: big.NewInt(1000),
			MaxCollateralPos:  big.NewInt(5000),
			MaxUniSwappable:   big.NewInt(3000),
		},
	}
}

func newPopulatedStore() (*MarketStore, [32]byte) {
	id := newMarketID(0x01)
	market := newMarket()
	store := &MarketStore{
		mu:      sync.RWMutex{},
		markets: map[[32]byte]*Market{id: market},
	}
	return store, id
}

// ── MarketStore ───────────────────────────────────────────────────────────────

func TestIds_ExcludesCanceled(t *testing.T) {
	store, id := newPopulatedStore()

	canceledID := newMarketID(0x02)
	store.markets[canceledID] = &Market{Canceled: true, Sorted: make([]*BorrowPosition, 0)}

	ids := store.Ids()
	if len(ids) != 1 {
		t.Fatalf("expected 1 active id, got %d", len(ids))
	}
	if ids[0] != id {
		t.Errorf("expected id %x, got %x", id, ids[0])
	}
}

func TestUpdate_ModifiesMarket(t *testing.T) {
	store, id := newPopulatedStore()

	store.Update(id, func(m *Market) {
		m.LLTV = big.NewInt(9e17)
	})

	store.mu.RLock()
	lltv := store.markets[id].LLTV
	store.mu.RUnlock()

	if lltv.Cmp(big.NewInt(9e17)) != 0 {
		t.Errorf("expected LLTV 9e17, got %s", lltv)
	}
}

func TestUpdate_NilMarket_NoOp(t *testing.T) {
	store, _ := newPopulatedStore()
	missingID := newMarketID(0xFF)

	// should not panic
	store.Update(missingID, func(m *Market) {
		t.Error("fn should not be called for nil market")
	})
}

func TestUpsert_ReplacesMarket(t *testing.T) {
	store, id := newPopulatedStore()

	newMkt := newMarket()
	newMkt.LLTV = big.NewInt(5e17)
	store.Upsert(id, newMkt)

	store.mu.RLock()
	lltv := store.markets[id].LLTV
	store.mu.RUnlock()

	if lltv.Cmp(big.NewInt(5e17)) != 0 {
		t.Errorf("expected replaced LLTV 5e17, got %s", lltv)
	}
}

func TestAllPosLen(t *testing.T) {
	store, id := newPopulatedStore()

	addr1 := common.HexToAddress("0x1")
	addr2 := common.HexToAddress("0x2")

	store.Update(id, func(m *Market) {
		m.Sorted = append(m.Sorted,
			newBorrowPosition(addr1, 9e17, 100),
			newBorrowPosition(addr2, 8e17, 200),
		)
	})

	if n := store.AllPosLen(); n != 2 {
		t.Errorf("expected 2 positions, got %d", n)
	}
}

// ── GetSnapshot ───────────────────────────────────────────────────────────────

func TestGetSnapshot_ReturnsNilForMissingMarket(t *testing.T) {
	store, _ := newPopulatedStore()
	snap := store.GetSnapshot(newMarketID(0xFF))
	if snap != nil {
		t.Error("expected nil snapshot for missing market")
	}
}

func TestGetSnapshot_ReturnsNilForCanceled(t *testing.T) {
	store, id := newPopulatedStore()
	store.Update(id, func(m *Market) { m.Canceled = true })

	snap := store.GetSnapshot(id)
	if snap != nil {
		t.Error("expected nil snapshot for canceled market")
	}
}

func TestGetSnapshot_ReturnsNilWhenStatsIncomplete(t *testing.T) {
	store, id := newPopulatedStore()
	store.Update(id, func(m *Market) { m.Stats.TotalBorrowAssets = nil })

	snap := store.GetSnapshot(id)
	if snap != nil {
		t.Error("expected nil snapshot when TotalBorrowAssets is nil")
	}
}

func TestGetSnapshot_CopiesPositions(t *testing.T) {
	store, id := newPopulatedStore()

	addr := common.HexToAddress("0x1")
	store.Update(id, func(m *Market) {
		m.Sorted = append(m.Sorted, newBorrowPosition(addr, 9e17, 100))
		m.ActiveLimit = 1
	})

	snap := store.GetSnapshot(id)
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if len(snap.Positions) != 1 {
		t.Errorf("expected 1 position in snapshot, got %d", len(snap.Positions))
	}
}

func TestGetSnapshot_IsolatesFromOriginal(t *testing.T) {
	store, id := newPopulatedStore()

	addr := common.HexToAddress("0x1")
	store.Update(id, func(m *Market) {
		m.Sorted = append(m.Sorted, newBorrowPosition(addr, 9e17, 100))
	})

	snap := store.GetSnapshot(id)
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}

	// Mutating snapshot must not affect original
	snap.Oracle.Price.SetInt64(0)

	store.mu.RLock()
	originalPrice := store.markets[id].Oracle.Price
	store.mu.RUnlock()

	if originalPrice.Sign() == 0 {
		t.Error("snapshot mutation leaked into original market")
	}
}

// ── ActiveLimit / hot zone ────────────────────────────────────────────────────

func TestActiveLimitFiltersHotZone(t *testing.T) {
	store, id := newPopulatedStore()

	HFThreshold := big.NewInt(11e17) // 1.1e18

	store.Update(id, func(m *Market) {
		// positions triées HF asc
		m.Sorted = []*BorrowPosition{
			newBorrowPosition(common.HexToAddress("0x1"), 8e17, 100),  // liquidable
			newBorrowPosition(common.HexToAddress("0x2"), 95e16, 100), // hot
			newBorrowPosition(common.HexToAddress("0x3"), 12e17, 100), // cold
			newBorrowPosition(common.HexToAddress("0x4"), 15e17, 100), // cold
		}
		// calculer ActiveLimit
		for i, p := range m.Sorted {
			if p.CachedHF.Cmp(HFThreshold) >= 0 {
				m.ActiveLimit = i
				return
			}
		}
		m.ActiveLimit = len(m.Sorted)
	})

	store.mu.RLock()
	m := store.markets[id]
	store.mu.RUnlock()

	m.Mu.RLock()
	hotZone := m.Sorted[:m.ActiveLimit]
	m.Mu.RUnlock()

	if len(hotZone) != 2 {
		t.Errorf("expected 2 positions in hot zone, got %d", len(hotZone))
	}
}

// ── MarketSnapshot helpers ────────────────────────────────────────────────────

func TestGetFirstHF_EmptyPositions(t *testing.T) {
	snap := &MarketSnapshot{Positions: []BorrowPosition{}}
	hf := snap.GetFirstHF()
	if hf.Sign() != 0 {
		t.Errorf("expected 0 for empty positions, got %s", hf)
	}
}

func TestGetFirstHF_ReturnsLowest(t *testing.T) {
	snap := &MarketSnapshot{
		Positions: []BorrowPosition{
			{CachedHF: big.NewInt(8e17)},
			{CachedHF: big.NewInt(9e17)},
		},
	}
	hf := snap.GetFirstHF()
	if hf.Cmp(big.NewInt(8e17)) != 0 {
		t.Errorf("expected 8e17, got %s", hf)
	}
}

// ── Concurrency ───────────────────────────────────────────────────────────────

func TestConcurrentUpdates_NoRace(t *testing.T) {
	store, id := newPopulatedStore()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int64) {
			defer wg.Done()
			store.Update(id, func(m *Market) {
				m.Stats.TotalBorrowAssets = big.NewInt(n * 1000)
			})
		}(int64(i))
	}
	wg.Wait()
}

func TestConcurrentGetSnapshot_NoRace(t *testing.T) {
	store, id := newPopulatedStore()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.GetSnapshot(id)
		}()
	}
	wg.Wait()
}