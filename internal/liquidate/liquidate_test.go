package liquidate

import (
	"math/big"
	"testing"

	"github.com/Stupnikjs/liquid/internal/cache"
)

// helpers
func bigInt(n int64) *big.Int { return big.NewInt(n) }

func baseSnap() cache.MarketSnapshot {
	return cache.MarketSnapshot{
		Stats: cache.MarketStats{
			TotalBorrowAssets: bigInt(1_000_000),
			TotalBorrowShares: bigInt(1_000_000),
			MaxUniSwappable:   bigInt(999_999_999),
		},
		LLTV:   bigInt(860_000_000_000_000_000), // 86%
		Oracle: cache.Oracle{Price: bigInt(1e18)},
	}
}

func basePos() *cache.BorrowPosition {
	return &cache.BorrowPosition{
		BorrowShares: bigInt(1000),
		MarketID:     [32]byte{1},
	}
}

func baseConsumer() *Consumer {
	return &Consumer{} // pas de RPC nécessaire
}

// SeizeAssets et MinOut doivent être positifs sur une position normale
func TestComputeAmounts_HappyPath(t *testing.T) {
	c := baseConsumer()
	l := c.ComputeAmounts(basePos())

	if l.SimErr != nil {
		t.Fatalf("unexpected SimErr: %v", l.SimErr)
	}
	if l.SeizeAssets == nil || l.SeizeAssets.Sign() <= 0 {
		t.Error("SeizeAssets should be > 0")
	}
	if l.MinOut == nil || l.MinOut.Sign() <= 0 {
		t.Error("MinOut should be > 0")
	}
	if l.RepayShares == nil || l.RepayShares.Sign() <= 0 {
		t.Error("RepayShares should be > 0")
	}
}

// SeizeAssets est cappé à MaxUniSwappable
func TestComputeAmounts_CappedByMaxUniSwappable(t *testing.T) {
	c := baseConsumer()
	snap := baseSnap()
	snap.Stats.MaxUniSwappable = bigInt(1) // cap très bas

	l := c.ComputeAmounts(basePos())

	if l.SimErr != nil {
		t.Fatalf("unexpected SimErr: %v", l.SimErr)
	}
	if l.SeizeAssets.Cmp(bigInt(1)) != 0 {
		t.Errorf("expected SeizeAssets=1, got %s", l.SeizeAssets)
	}
}

// BorrowShares=0 → montants nuls
func TestComputeAmounts_ZeroBorrowShares(t *testing.T) {
	c := baseConsumer()
	pos := basePos()
	pos.BorrowShares = bigInt(0)

	l := c.ComputeAmounts(pos)

	if l.SimErr != nil {
		t.Fatalf("unexpected SimErr: %v", l.SimErr)
	}
	if l.SeizeAssets.Sign() != 0 {
		t.Errorf("expected SeizeAssets=0, got %s", l.SeizeAssets)
	}
}
