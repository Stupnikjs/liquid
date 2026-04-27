package cache

import (
	"testing"
)

// ── HF ───────────────────────────────────────────────────────────────────────

func TestHF_NilWhenBorrowSharesZero(t *testing.T) {
	m := newMarket()
	p := newPos(testAddrA, 1000, 0, nil)

	if hfSign(m.HF(p)) != 0 {
		t.Error("expected nil HF when BorrowShares == 0")
	}
}

func TestHF_NilWhenCollateralZero(t *testing.T) {
	m := newMarket()
	p := newPos(testAddrA, 0, 500, nil)

	if hfSign(m.HF(p)) != 0 {
		t.Error("expected nil HF when CollateralAssets == 0")
	}
}

func TestHF_NilWhenStatsNil(t *testing.T) {
	m := newMarketCustom(testPrice, testLLTV, nil, nil)
	p := newPos(testAddrA, 1000, 500, nil)

	if hfSign(m.HF(p)) != 0 {
		t.Error("expected nil HF when TotalBorrowAssets / TotalBorrowShares are nil")
	}
}

// Healthy: collateral=2000, borrow=1000, price=1e18, lltv=8e17
// numerator   = 2000 * 1e18 * 8e17 = 1.6e39
// denominator = 1000 * 1e36        = 1e39
// HF (WAD)    = 1.6e39 / 1e39 * 1e18 → positive
func TestHF_PositiveForHealthyPosition(t *testing.T) {
	m := newMarket()
	p := newPos(testAddrA, 2000, 1000, nil)

	hf := m.HF(p)
	if hf == nil || hf.Sign() <= 0 {
		t.Errorf("expected positive HF for healthy position, got %v", hf)
	}
}

func TestHF_RiskyLowerThanHealthy(t *testing.T) {
	m := newMarket()
	healthy := newPos(testAddrA, 2000, 1000, nil)
	risky := newPos(testAddrB, 500, 1000, nil)

	hfH := m.HF(healthy)
	hfR := m.HF(risky)

	if hfH == nil || hfR == nil {
		t.Fatal("neither HF should be nil")
	}
	if hfR.Cmp(hfH) >= 0 {
		t.Errorf("risky HF (%s) should be < healthy HF (%s)", hfR, hfH)
	}
}

// ── RecomputeHFUnsafe ─────────────────────────────────────────────────────────

func TestRecomputeHFUnsafe_UpdatesCachedHF(t *testing.T) {
	m := newMarket()
	p := newPos(testAddrA, 2000, 1000, wi(0))
	m.Positions = []*BorrowPosition{p}

	m.RecomputeHFUnsafe(1)

	if p.CachedHF == nil || p.CachedHF.Sign() <= 0 {
		t.Errorf("CachedHF should be > 0 after recompute, got %v", p.CachedHF)
	}
}

// RecomputeHFUnsafe(n) updates indices [0, n): p1 updated, p2 untouched.
func TestRecomputeHFUnsafe_RespectsNLimit(t *testing.T) {
	m := newMarket()
	p1 := newPos(testAddrA, 2000, 1000, wi(0))
	p2 := newPos(testAddrB, 2000, 1000, wi(0))
	m.Positions = []*BorrowPosition{p1, p2}

	m.RecomputeHFUnsafe(1)

	if p1.CachedHF == nil || p1.CachedHF.Sign() <= 0 {
		t.Errorf("p1 should be updated, got %v", p1.CachedHF)
	}
	if p2.CachedHF == nil || p2.CachedHF.Sign() != 0 {
		t.Errorf("p2 should NOT be updated, got %v", p2.CachedHF)
	}
}

// ActiveIndex=1 means only p1 is in the hot zone; recomputing 1 entry
// must leave p2 untouched regardless of ActiveIndex.
func TestRecomputeHFUnsafe_UsesActiveIndex(t *testing.T) {
	m := newMarket()
	p1 := newPos(testAddrA, 2000, 1000, wi(0))
	p2 := newPos(testAddrB, 2000, 1000, wi(0))
	m.Positions = []*BorrowPosition{p1, p2}
	m.ActiveIndex = 1

	m.RecomputeHFUnsafe(m.ActiveIndex)

	if p1.CachedHF == nil || p1.CachedHF.Sign() <= 0 {
		t.Errorf("p1 (active) should be recomputed, got %v", p1.CachedHF)
	}
	if p2.CachedHF == nil || p2.CachedHF.Sign() != 0 {
		t.Errorf("p2 (cold) should not be recomputed, got %v", p2.CachedHF)
	}
}

// ── SortAllPositionsByHFUnsafe ────────────────────────────────────────────────

func TestSortAllPositionsByHFUnsafe_SortsAscending(t *testing.T) {
	m := newMarket()
	m.Positions = []*BorrowPosition{
		newPosHF(testAddrA, wi(15e17)),
		newPosHF(testAddrB, wi(8e17)),
		newPosHF(testAddrC, wi(11e17)),
	}

	m.SortAllPositionsByHFUnsafe()

	want := []int64{8e17, 11e17, 15e17}
	for i, w := range want {
		got := m.Positions[i].CachedHF
		if got == nil || got.Cmp(wi(w)) != 0 {
			t.Errorf("pos[%d]: want %d, got %v", i, w, got)
		}
	}
}

func TestSortAllPositionsByHFUnsafe_NilCachedHFLast(t *testing.T) {
	m := newMarket()
	m.Positions = []*BorrowPosition{
		newPosHF(testAddrA, nil),
		newPosHF(testAddrB, wi(9e17)),
	}

	m.SortAllPositionsByHFUnsafe()

	if m.Positions[0].CachedHF == nil {
		t.Error("nil CachedHF should be sorted last")
	}
	if m.Positions[1].CachedHF != nil {
		t.Error("nil CachedHF should be at index 1")
	}
}

func TestSortAllPositionsByHFUnsafe_EmptySlice_NoPanic(t *testing.T) {
	m := newMarket()
	m.SortAllPositionsByHFUnsafe() // must not panic
}

// ── MarketSnapshot.GetFirstHF ─────────────────────────────────────────────────

func TestGetFirstHF_EmptySnapshot(t *testing.T) {
	snap := newSnapshot([]BorrowPosition{})
	if snap.GetFirstHF() != nil {
		t.Error("expected nil for empty positions")
	}
}

func TestGetFirstHF_ReturnsLowestAfterSort(t *testing.T) {
	snap := newSnapshot([]BorrowPosition{
		{CachedHF: wi(8e17)},
		{CachedHF: wi(12e17)},
	})
	hf := snap.GetFirstHF()
	if hf == nil || hf.Cmp(wi(8e17)) != 0 {
		t.Errorf("expected 8e17, got %v", hf)
	}
}

func TestGetFirstHF_SkipsLeadingNilEntries(t *testing.T) {
	// After sort, non-nil HFs come first; nil entries trail.
	// This snapshot is already in the correct post-sort order.
	snap := newSnapshot([]BorrowPosition{
		{CachedHF: wi(9e17)},
		{CachedHF: nil},
	})
	hf := snap.GetFirstHF()
	if hf == nil || hf.Cmp(wi(9e17)) != 0 {
		t.Errorf("expected 9e17, got %v", hf)
	}
}

// Edge: all entries have nil HF — GetFirstHF must return nil, not panic.
func TestGetFirstHF_AllNil(t *testing.T) {
	snap := newSnapshot([]BorrowPosition{
		{CachedHF: nil},
		{CachedHF: nil},
	})
	if snap.GetFirstHF() != nil {
		t.Error("expected nil when all CachedHFs are nil")
	}
}
