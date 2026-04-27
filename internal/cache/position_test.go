package cache

import (
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// ── InsertPositionUnsafe ──────────────────────────────────────────────────────

func TestInsertPositionUnsafe_EmptySlice(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(100)))

	if len(m.Positions) != 1 || m.Positions[0].Address != testAddrA {
		t.Fatalf("unexpected state after insert: %v", m.Positions)
	}
}

func TestInsertPositionUnsafe_MaintainsAscendingOrder(t *testing.T) {
	m := newMarket()
	// Insert in reverse HF order; slice must always be sorted ascending.
	m.InsertPositionUnsafe(newPosHF(testAddrC, wi(300)))
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(100)))
	m.InsertPositionUnsafe(newPosHF(testAddrB, wi(200)))

	want := []common.Address{testAddrA, testAddrB, testAddrC}
	got := posAddrs(m)
	for i, w := range want {
		if got[i] != w {
			t.Errorf("pos[%d]: want %v, got %v", i, w, got[i])
		}
	}
}

func TestInsertPositionUnsafe_NilHFAppendedAtEnd(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(100)))
	m.InsertPositionUnsafe(newPosHF(testAddrB, nil))

	if m.Positions[0].Address != testAddrA {
		t.Error("non-nil HF should come first")
	}
	if m.Positions[1].Address != testAddrB {
		t.Error("nil HF should be last")
	}
}

func TestInsertPositionUnsafe_AllNilHFs(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrA, nil))
	m.InsertPositionUnsafe(newPosHF(testAddrB, nil))

	if len(m.Positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(m.Positions))
	}
}

func TestInsertPositionUnsafe_InsertAtFront(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrB, wi(200)))
	m.InsertPositionUnsafe(newPosHF(testAddrC, wi(300)))
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(50)))

	if m.Positions[0].Address != testAddrA {
		t.Errorf("lowest HF should be at index 0, got %v", m.Positions[0].Address)
	}
}

func TestInsertPositionUnsafe_DuplicateHF_BothPresent(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(100)))
	m.InsertPositionUnsafe(newPosHF(testAddrB, wi(100)))

	if len(m.Positions) != 2 {
		t.Fatalf("expected 2 positions for duplicate HF, got %d", len(m.Positions))
	}
}

// ── RemovePositionUnsafe ──────────────────────────────────────────────────────

func TestRemovePositionUnsafe_RemovesMiddle(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(100)))
	m.InsertPositionUnsafe(newPosHF(testAddrB, wi(200)))
	m.InsertPositionUnsafe(newPosHF(testAddrC, wi(300)))

	m.RemovePositionUnsafe(testAddrB)

	if len(m.Positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(m.Positions))
	}
	for _, p := range m.Positions {
		if p.Address == testAddrB {
			t.Error("testAddrB should have been removed")
		}
	}
}

func TestRemovePositionUnsafe_RemovesFirst(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(100)))
	m.InsertPositionUnsafe(newPosHF(testAddrB, wi(200)))

	m.RemovePositionUnsafe(testAddrA)

	if len(m.Positions) != 1 || m.Positions[0].Address != testAddrB {
		t.Error("only testAddrB should remain after removing testAddrA")
	}
}

func TestRemovePositionUnsafe_RemovesLast(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(100)))
	m.InsertPositionUnsafe(newPosHF(testAddrB, wi(200)))

	m.RemovePositionUnsafe(testAddrB)

	if len(m.Positions) != 1 || m.Positions[0].Address != testAddrA {
		t.Error("only testAddrA should remain after removing testAddrB")
	}
}

func TestRemovePositionUnsafe_MissingAddress_NoOp(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(100)))

	m.RemovePositionUnsafe(testAddrD)

	if len(m.Positions) != 1 {
		t.Error("slice should be unchanged when address not found")
	}
}

func TestRemovePositionUnsafe_EmptySlice_NoPanic(t *testing.T) {
	m := newMarket()
	m.RemovePositionUnsafe(testAddrA) // must not panic
}

// ── UpdatePositionUnsafe ──────────────────────────────────────────────────────

func TestUpdatePositionUnsafe_HFIncreaseMovesToBack(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(100)))
	m.InsertPositionUnsafe(newPosHF(testAddrB, wi(200)))
	m.InsertPositionUnsafe(newPosHF(testAddrC, wi(300)))

	updated := newPosHF(testAddrA, wi(999))
	m.UpdatePositionUnsafe(updated)

	if len(m.Positions) != 3 {
		t.Fatalf("expected 3 positions after update, got %d", len(m.Positions))
	}
	if m.Positions[2].Address != testAddrA {
		t.Errorf("testAddrA should be last after HF increase, got %v", m.Positions[2].Address)
	}
}

// UpdatePositionUnsafe = Remove (no-op) + Insert, so an unknown address is
// silently inserted. This test documents that implicit behaviour.
func TestUpdatePositionUnsafe_UnknownAddress_Inserts(t *testing.T) {
	m := newMarket()
	pA := newPos(testAddrA, 2000, 1000, nil)
	pA.CachedHF = m.HF(pA)
	m.InsertPositionUnsafe(pA)

	// testAddrD: lower collateral ratio → lower HF than testAddrA
	unknown := newPos(testAddrD, 500, 1000, nil)
	m.UpdatePositionUnsafe(unknown)

	if len(m.Positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(m.Positions))
	}
	if m.Positions[0].Address != testAddrD {
		t.Errorf("testAddrD (lower HF) should be first, got %v", m.Positions[0].Address)
	}
}

// ── GetBorrowPosition ─────────────────────────────────────────────────────────

func TestGetBorrowPosition_Found(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(100)))
	m.InsertPositionUnsafe(newPosHF(testAddrB, wi(200)))

	p := m.GetBorrowPosition(testAddrB)
	if p == nil || p.Address != testAddrB {
		t.Errorf("expected testAddrB, got %v", p)
	}
}

func TestGetBorrowPosition_NotFound(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(100)))

	if p := m.GetBorrowPosition(testAddrD); p != nil {
		t.Errorf("expected nil for unknown address, got %v", p)
	}
}

func TestGetBorrowPosition_EmptyMarket(t *testing.T) {
	m := newMarket()
	if p := m.GetBorrowPosition(testAddrA); p != nil {
		t.Errorf("expected nil on empty market, got %v", p)
	}
}

// ── Order invariant ───────────────────────────────────────────────────────────

func TestPositionsRemainSortedAfterMixedOps(t *testing.T) {
	m := newMarket()

	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(300)))
	m.InsertPositionUnsafe(newPosHF(testAddrB, wi(100)))
	m.InsertPositionUnsafe(newPosHF(testAddrC, wi(200)))
	m.RemovePositionUnsafe(testAddrA)
	m.InsertPositionUnsafe(newPosHF(testAddrD, wi(150)))
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(50)))

	hfs := posHFs(m)
	for i := 1; i < len(hfs); i++ {
		if hfs[i-1] == nil {
			continue
		}
		if hfs[i] != nil && hfs[i-1].Cmp(hfs[i]) > 0 {
			t.Errorf("out of order at index %d: %v > %v", i, hfs[i-1], hfs[i])
		}
	}
}

// ── Concurrency ───────────────────────────────────────────────────────────────

func TestInsertPosition_Concurrent_NoRace(t *testing.T) {
	m := newMarket()
	addrs := []common.Address{testAddrA, testAddrB, testAddrC, testAddrD}

	var wg sync.WaitGroup
	for _, addr := range addrs {
		wg.Add(1)
		go func(a common.Address) {
			defer wg.Done()
			m.InsertPosition(newPosHF(a, nil))
		}(addr)
	}
	wg.Wait()

	seen := make(map[common.Address]int)
	m.Mu.RLock()
	for _, p := range m.Positions {
		seen[p.Address]++
	}
	m.Mu.RUnlock()

	for _, addr := range addrs {
		if seen[addr] != 1 {
			t.Errorf("address %v appears %d times, want 1", addr, seen[addr])
		}
	}
}

func TestGetBorrowPosition_ConcurrentReads_NoRace(t *testing.T) {
	m := newMarket()
	m.InsertPositionUnsafe(newPosHF(testAddrA, wi(100)))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if p := m.GetBorrowPosition(testAddrA); p == nil {
				t.Errorf("concurrent read returned nil")
			}
		}()
	}
	wg.Wait()
}
