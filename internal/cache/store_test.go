package cache

import (
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// ── MarketStore.Ids ───────────────────────────────────────────────────────────

func TestIds_ExcludesCanceled(t *testing.T) {
	store, id := newStore()

	canceledID := newMarketID(0x02)
	store.markets[canceledID] = &Market{Canceled: true, Positions: make([]*BorrowPosition, 0)}

	ids := store.Ids()
	if len(ids) != 1 {
		t.Fatalf("expected 1 active id, got %d", len(ids))
	}
	if ids[0] != id {
		t.Errorf("expected id %x, got %x", id, ids[0])
	}
}

// ── MarketStore.Update ────────────────────────────────────────────────────────

func TestUpdate_ModifiesMarket(t *testing.T) {
	store, id := newStore()

	store.Update(id, func(m *Market) {
		m.LLTV = wi(9e17)
	})

	store.mu.RLock()
	lltv := store.markets[id].LLTV
	store.mu.RUnlock()

	if lltv.Cmp(wi(9e17)) != 0 {
		t.Errorf("expected LLTV 9e17, got %s", lltv)
	}
}

func TestUpdate_MissingID_NoOp(t *testing.T) {
	store, _ := newStore()

	// fn must never be called for an id that does not exist in the store.
	store.Update(newMarketID(0xFF), func(m *Market) {
		t.Error("fn should not be called for missing market")
	})
}

// ── MarketStore.Upsert ────────────────────────────────────────────────────────

func TestUpsert_ReplacesExistingMarket(t *testing.T) {
	store, id := newStore()

	replacement := newMarket()
	replacement.LLTV = wi(5e17)
	store.Upsert(id, replacement)

	store.mu.RLock()
	lltv := store.markets[id].LLTV
	store.mu.RUnlock()

	if lltv.Cmp(wi(5e17)) != 0 {
		t.Errorf("expected LLTV 5e17 after upsert, got %s", lltv)
	}
}

func TestUpsert_InsertsNewMarket(t *testing.T) {
	store, _ := newStore()

	newID := newMarketID(0x02)
	store.Upsert(newID, newMarket())

	store.mu.RLock()
	_, exists := store.markets[newID]
	store.mu.RUnlock()

	if !exists {
		t.Error("expected new market to be inserted by Upsert")
	}
}

// ── MarketStore.AllPosLen ─────────────────────────────────────────────────────

func TestAllPosLen_SumsAcrossMarkets(t *testing.T) {
	store, id := newStore()

	store.Update(id, func(m *Market) {
		m.Positions = append(m.Positions,
			newPosHF(common.HexToAddress("0x1"), wi(9e17)),
			newPosHF(common.HexToAddress("0x2"), wi(8e17)),
		)
	})

	if n := store.AllPosLen(); n != 2 {
		t.Errorf("expected 2 positions, got %d", n)
	}
}

// ── MarketStore.GetSnapshot ───────────────────────────────────────────────────

func TestGetSnapshot_NilForMissingMarket(t *testing.T) {
	store, _ := newStore()
	if snap := store.GetSnapshot(newMarketID(0xFF)); snap != nil {
		t.Error("expected nil snapshot for missing market")
	}
}

func TestGetSnapshot_NilForCanceledMarket(t *testing.T) {
	store, id := newStore()
	store.Update(id, func(m *Market) { m.Canceled = true })

	if snap := store.GetSnapshot(id); snap != nil {
		t.Error("expected nil snapshot for canceled market")
	}
}

func TestGetSnapshot_NilWhenTotalBorrowAssetsNil(t *testing.T) {
	store, id := newStore()
	store.Update(id, func(m *Market) { m.Stats.TotalBorrowAssets = nil })

	if snap := store.GetSnapshot(id); snap != nil {
		t.Error("expected nil snapshot when TotalBorrowAssets is nil")
	}
}

func TestGetSnapshot_CopiesPositions(t *testing.T) {
	store, id := newStore()
	store.Update(id, func(m *Market) {
		m.Positions = append(m.Positions, newPosHF(testAddrA, wi(9e17)))
		m.ActiveIndex = 1
	})

	snap := store.GetSnapshot(id)
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if len(snap.Positions) != 1 {
		t.Errorf("expected 1 position in snapshot, got %d", len(snap.Positions))
	}
}

// Mutating the snapshot's Oracle.Price must not affect the original market.
func TestGetSnapshot_IsolatesFromOriginal(t *testing.T) {
	store, id := newStore()
	store.Update(id, func(m *Market) {
		m.Positions = append(m.Positions, newPosHF(testAddrA, wi(9e17)))
		m.ActiveIndex = 1
	})

	snap := store.GetSnapshot(id)
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	snap.Oracle.Price.SetInt64(0)

	store.mu.RLock()
	originalPrice := store.markets[id].Oracle.Price
	store.mu.RUnlock()

	if originalPrice.Sign() == 0 {
		t.Error("snapshot mutation leaked into original market Oracle.Price")
	}
}

// ── Hot zone / ActiveIndex ────────────────────────────────────────────────────

func TestActiveLimitFiltersHotZone(t *testing.T) {
	store, id := newStore()
	hfThreshold := wi(11e17)

	store.Update(id, func(m *Market) {
		m.Positions = []*BorrowPosition{
			newPosHF(common.HexToAddress("0x1"), wi(8e17)),  // liquidable
			newPosHF(common.HexToAddress("0x2"), wi(95e16)), // hot
			newPosHF(common.HexToAddress("0x3"), wi(12e17)), // cold
			newPosHF(common.HexToAddress("0x4"), wi(15e17)), // cold
		}
		m.ActiveIndex = len(m.Positions) // default: all cold
		for i, p := range m.Positions {
			if p.CachedHF.Cmp(hfThreshold) >= 0 {
				m.ActiveIndex = i
				return
			}
		}
	})

	store.mu.RLock()
	m := store.markets[id]
	store.mu.RUnlock()

	m.Mu.RLock()
	hotZone := m.Positions[:m.ActiveIndex]
	m.Mu.RUnlock()

	if len(hotZone) != 2 {
		t.Errorf("expected 2 positions in hot zone, got %d", len(hotZone))
	}
}

// ── Concurrency ───────────────────────────────────────────────────────────────

func TestConcurrentUpdates_NoRace(t *testing.T) {
	store, id := newStore()

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
	store, id := newStore()

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
