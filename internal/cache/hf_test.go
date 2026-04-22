package cache

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// WAD = 1e18
var WAD = big.NewInt(1e18)

// marketWithStats construit un Market minimal pour calculer le HF.
// price et lltv en WAD (1e18 = 1.0)
func marketWithStats(price, lltv, totalBorrowAssets, totalBorrowShares *big.Int) *Market {
	return &Market{
		Oracle: Oracle{Price: price},
		LLTV:   lltv,
		Stats: MarketStats{
			TotalBorrowAssets: totalBorrowAssets,
			TotalBorrowShares: totalBorrowShares,
		},
		Positions: make([]*BorrowPosition, 0),
	}
}

// pos construit une BorrowPosition simple.
func pos(addr string, collateral, borrowShares int64) *BorrowPosition {
	return &BorrowPosition{
		Address:         common.HexToAddress(addr),
		CollateralAssets: big.NewInt(collateral),
		BorrowShares:    big.NewInt(borrowShares),
	}
}

// ── HF ───────────────────────────────────────────────────────────────────────

func TestHF_ZeroWhenBorrowSharesZero(t *testing.T) {
	m := marketWithStats(
		big.NewInt(1e18), big.NewInt(8e17),
		big.NewInt(1000), big.NewInt(1000),
	)
	p := pos("0x1", 1000, 0) // BorrowShares = 0

	hf := m.HF(p)
	if hf.Sign() != 0 {
		t.Errorf("expected 0, got %s", hf)
	}
}

func TestHF_ZeroWhenCollateralZero(t *testing.T) {
	m := marketWithStats(
		big.NewInt(1e18), big.NewInt(8e17),
		big.NewInt(1000), big.NewInt(1000),
	)
	p := pos("0x1", 0, 500)

	hf := m.HF(p)
	if hf.Sign() != 0 {
		t.Errorf("expected 0, got %s", hf)
	}
}

func TestHF_ZeroWhenNilStats(t *testing.T) {
	m := marketWithStats(big.NewInt(1e18), big.NewInt(8e17), nil, nil)
	p := pos("0x1", 1000, 500)

	hf := m.HF(p)
	if hf.Sign() != 0 {
		t.Errorf("expected 0 for nil stats, got %s", hf)
	}
}

// Cas sain : collateral > borrow → HF > 1
// TotalBorrowAssets = TotalBorrowShares → borrowAssets = borrowShares
// HF = collateral * price * lltv / (borrowAssets * 1e36)
// Avec collateral=2000, price=1e18, lltv=8e17, borrow=1000 :
// numerator   = 2000 * 1e18 * 8e17 = 1.6e39
// denominator = 1000 * 1e36        = 1e39
// HF = 1.6e39 / 1e39 → 1 (integer div) mais > seuil 0
func TestHF_HealthyPosition(t *testing.T) {
	m := marketWithStats(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), // price = 1e18
		new(big.Int).Mul(big.NewInt(8), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)), // lltv = 8e17
		big.NewInt(1000),
		big.NewInt(1000),
	)
	p := pos("0x1", 2000, 1000)

	hf := m.HF(p)
	if hf.Sign() <= 0 {
		t.Errorf("expected positive HF for healthy position, got %s", hf)
	}
}

// Position risquée : collateral < borrow → HF < HF sain
func TestHF_RiskyLowerThanHealthy(t *testing.T) {
	price := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	lltv := new(big.Int).Mul(big.NewInt(8), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil))
	totalBA := big.NewInt(1000)
	totalBS := big.NewInt(1000)

	m := marketWithStats(price, lltv, totalBA, totalBS)

	healthy := pos("0x1", 2000, 1000)
	risky := pos("0x2", 500, 1000)

	hfHealthy := m.HF(healthy)
	hfRisky := m.HF(risky)

	if hfRisky.Cmp(hfHealthy) >= 0 {
		t.Errorf("risky HF (%s) should be < healthy HF (%s)", hfRisky, hfHealthy)
	}
}

// ── RecomputeHF ───────────────────────────────────────────────────────────────

func TestRecomputeHF_UpdatesCachedHF(t *testing.T) {
	m := marketWithStats(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
		new(big.Int).Mul(big.NewInt(8), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)),
		big.NewInt(1000),
		big.NewInt(1000),
	)
	p := pos("0x1", 2000, 1000)
	p.CachedHF = big.NewInt(0) // valeur initiale nulle
	m.Positions = []*BorrowPosition{p}

	m.RecomputeHF(1)

	if p.CachedHF.Sign() <= 0 {
		t.Errorf("CachedHF should be > 0 after recompute, got %s", p.CachedHF)
	}
}

func TestRecomputeHF_RespectsNLimit(t *testing.T) {
	m := marketWithStats(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
		new(big.Int).Mul(big.NewInt(8), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)),
		big.NewInt(1000),
		big.NewInt(1000),
	)
	p1 := pos("0x1", 2000, 1000)
	p2 := pos("0x2", 2000, 1000)
	p1.CachedHF = big.NewInt(0)
	p2.CachedHF = big.NewInt(0)
	m.Positions = []*BorrowPosition{p1, p2}

	m.RecomputeHF(1) // seulement la première

	if p1.CachedHF.Sign() <= 0 {
		t.Errorf("p1 CachedHF should be updated")
	}
	if p2.CachedHF.Sign() != 0 {
		t.Errorf("p2 CachedHF should not be updated, got %s", p2.CachedHF)
	}
}

func TestRecomputeActiveHF_UsesActiveIndex(t *testing.T) {
	m := marketWithStats(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
		new(big.Int).Mul(big.NewInt(8), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)),
		big.NewInt(1000),
		big.NewInt(1000),
	)
	p1 := pos("0x1", 2000, 1000)
	p2 := pos("0x2", 2000, 1000)
	p1.CachedHF = big.NewInt(0)
	p2.CachedHF = big.NewInt(0)
	m.Positions = []*BorrowPosition{p1, p2}
	m.ActiveIndex = 1 // seulement p1 en hot zone

	m.RecomputeActiveHF()

	if p1.CachedHF.Sign() <= 0 {
		t.Errorf("p1 (active) should be recomputed")
	}
	if p2.CachedHF.Sign() != 0 {
		t.Errorf("p2 (cold) should not be recomputed")
	}
}

// ── SortAllPositionsByHF ──────────────────────────────────────────────────────

func TestSortAllPositionsByHF_SortsAscending(t *testing.T) {
	m := marketWithStats(big.NewInt(1e18), big.NewInt(8e17), big.NewInt(1000), big.NewInt(1000))

	p1 := pos("0x1", 100, 100)
	p2 := pos("0x2", 100, 100)
	p3 := pos("0x3", 100, 100)
	p1.CachedHF = big.NewInt(15e17) // 1.5
	p2.CachedHF = big.NewInt(8e17)  // 0.8 → liquidable
	p3.CachedHF = big.NewInt(11e17) // 1.1
	m.Positions = []*BorrowPosition{p1, p2, p3}

	m.SortAllPositionsByHF()

	expected := []int64{8e17, 11e17, 15e17}
	for i, exp := range expected {
		if m.Positions[i].CachedHF.Cmp(big.NewInt(exp)) != 0 {
			t.Errorf("pos[%d]: expected %d, got %s", i, exp, m.Positions[i].CachedHF)
		}
	}
}

func TestSortAllPositionsByHF_NilCachedHFLast(t *testing.T) {
	m := marketWithStats(big.NewInt(1e18), big.NewInt(8e17), big.NewInt(1000), big.NewInt(1000))

	p1 := pos("0x1", 100, 100)
	p2 := pos("0x2", 100, 100)
	p1.CachedHF = nil             // nil → rejeté en fin
	p2.CachedHF = big.NewInt(9e17)
	m.Positions = []*BorrowPosition{p1, p2}

	m.SortAllPositionsByHF()

	if m.Positions[0].CachedHF == nil {
		t.Error("nil CachedHF should be sorted last, not first")
	}
	if m.Positions[1].CachedHF != nil {
		t.Error("nil CachedHF should be at the end")
	}
}

func TestSortAllPositionsByHF_Empty(t *testing.T) {
	m := marketWithStats(big.NewInt(1e18), big.NewInt(8e17), big.NewInt(1000), big.NewInt(1000))
	m.Positions = []*BorrowPosition{}

	// ne doit pas paniquer
	m.SortAllPositionsByHF()
}

// ── GetFirstHF ────────────────────────────────────────────────────────────────

func TestMarketGetFirstHF_Empty(t *testing.T) {
	m := marketWithStats(big.NewInt(1e18), big.NewInt(8e17), big.NewInt(1000), big.NewInt(1000))
	m.Positions = []*BorrowPosition{}

	hf := m.GetFirstHF()
	if hf.Sign() != 0 {
		t.Errorf("expected 0 for empty positions, got %s", hf)
	}
}

func TestMarketGetFirstHF_ReturnsLowest(t *testing.T) {
	m := marketWithStats(big.NewInt(1e18), big.NewInt(8e17), big.NewInt(1000), big.NewInt(1000))

	p1 := pos("0x1", 100, 100)
	p2 := pos("0x2", 100, 100)
	p1.CachedHF = big.NewInt(8e17)
	p2.CachedHF = big.NewInt(12e17)
	m.Positions = []*BorrowPosition{p1, p2}

	hf := m.GetFirstHF()
	if hf.Cmp(big.NewInt(8e17)) != 0 {
		t.Errorf("expected 8e17, got %s", hf)
	}
}