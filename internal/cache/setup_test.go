package cache

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// ── Constants ─────────────────────────────────────────────────────────────────

var (
	testAddrA = common.HexToAddress("0xAAAA")
	testAddrB = common.HexToAddress("0xBBBB")
	testAddrC = common.HexToAddress("0xCCCC")
	testAddrD = common.HexToAddress("0xDDDD")

	// Canonical WAD / LLTV values reused across test files.
	testPrice = exp10(18)            // 1e18 — 1:1 oracle price
	testLLTV  = mulInt(8, exp10(17)) // 8e17 — 80% LTV
)

// ── Big-int shorthands ────────────────────────────────────────────────────────

// wi ("whole int") wraps a plain int64 in a *big.Int.
func wi(n int64) *big.Int { return big.NewInt(n) }

// exp10 returns 10^n as a new *big.Int.
func exp10(n int64) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(n), nil)
}

// mulInt returns a * 10^0 scaled by factor — convenience for e.g. 8 * 1e17.
func mulInt(factor int64, base *big.Int) *big.Int {
	return new(big.Int).Mul(big.NewInt(factor), base)
}

// ── Position constructors ─────────────────────────────────────────────────────

// newPos builds a BorrowPosition with explicit collateral, borrow shares and
// an optional pre-set CachedHF.  Pass nil for cachedHF when the HF should be
// computed later by the market.
func newPos(addr common.Address, collateral, borrowShares int64, cachedHF *big.Int) *BorrowPosition {
	return &BorrowPosition{
		Address:          addr,
		CollateralAssets: big.NewInt(collateral),
		BorrowShares:     big.NewInt(borrowShares),
		CachedHF:         cachedHF,
	}
}

// newPosHF builds a BorrowPosition whose only meaningful field is CachedHF.
// Use for sort / insert tests that do not exercise HF computation.
func newPosHF(addr common.Address, cachedHF *big.Int) *BorrowPosition {
	return newPos(addr, 0, 0, cachedHF)
}

// ── Market constructors ───────────────────────────────────────────────────────

// newMarket returns a fully-initialised *Market using the canonical test
// price and LLTV so every file uses the same baseline.
func newMarket() *Market {
	return &Market{
		Oracle:    Oracle{Price: new(big.Int).Set(testPrice)},
		LLTV:      new(big.Int).Set(testLLTV),
		Positions: make([]*BorrowPosition, 0),
		Stats: MarketStats{
			TotalBorrowAssets: big.NewInt(1000),
			TotalBorrowShares: big.NewInt(1000),
			MaxCollateralPos:  big.NewInt(5000),
			MaxUniSwappable:   big.NewInt(3000),
		},
	}
}

// newMarketCustom allows overriding price / lltv / totalBorrowAssets /
// totalBorrowShares when a test needs non-default values.
func newMarketCustom(price, lltv, totalBorrowAssets, totalBorrowShares *big.Int) *Market {
	m := newMarket()
	m.Oracle.Price = price
	m.LLTV = lltv
	m.Stats.TotalBorrowAssets = totalBorrowAssets
	m.Stats.TotalBorrowShares = totalBorrowShares
	return m
}

// ── MarketStore constructor ───────────────────────────────────────────────────

func newMarketID(b byte) [32]byte {
	var id [32]byte
	id[0] = b
	return id
}

// newStore returns a *MarketStore seeded with one active market under id 0x01.
func newStore() (*MarketStore, [32]byte) {
	id := newMarketID(0x01)
	store := &MarketStore{
		mu:      sync.RWMutex{},
		markets: map[[32]byte]*Market{id: newMarket()},
	}
	return store, id
}

// ── Snapshot helper ───────────────────────────────────────────────────────────

func newSnapshot(positions []BorrowPosition) *MarketSnapshot {
	return &MarketSnapshot{Positions: positions}
}

// ── HF sign helper ────────────────────────────────────────────────────────────

// hfSign treats nil as zero, mirroring HF()'s nil-not-zero contract.
func hfSign(hf *big.Int) int {
	if hf == nil {
		return 0
	}
	return hf.Sign()
}

// ── Slice inspection helpers ──────────────────────────────────────────────────

func posAddrs(m *Market) []common.Address {
	out := make([]common.Address, len(m.Positions))
	for i, p := range m.Positions {
		out[i] = p.Address
	}
	return out
}

func posHFs(m *Market) []*big.Int {
	out := make([]*big.Int, len(m.Positions))
	for i, p := range m.Positions {
		out[i] = p.CachedHF
	}
	return out
}
