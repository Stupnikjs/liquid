package onchain

import (
	"math/big"
	"sync"
	"testing"

	market "github.com/Stupnikjs/morpho-sepolia/internal/cache"
	"github.com/ethereum/go-ethereum/common"
)

// ── mock MarketReader ─────────────────────────────────────────────────────────

type mockReader struct {
	mu      sync.RWMutex
	markets map[[32]byte]*market.Market
}

func newMockReader() *mockReader {
	return &mockReader{
		markets: make(map[[32]byte]*market.Market),
	}
}

func (r *mockReader) Update(id [32]byte, fn func(m *market.Market)) {
	r.mu.Lock()
	m := r.markets[id]
	r.mu.Unlock()
	if m == nil {
		return
	}
	m.Mu.Lock()
	fn(m)
	m.Mu.Unlock()
}

func (r *mockReader) set(id [32]byte, m *market.Market) {
	r.mu.Lock()
	r.markets[id] = m
	r.mu.Unlock()
}

func (r *mockReader) get(id [32]byte) *market.Market {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.markets[id]
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newMarketID(b byte) [32]byte {
	var id [32]byte
	id[0] = b
	return id
}

func emptyMarket() *market.Market {
	return &market.Market{
		Positions: make([]*market.BorrowPosition, 0),
		Stats:     market.MarketStats{},
		Oracle:    market.Oracle{},
	}
}

// ── ApplyResults ──────────────────────────────────────────────────────────────

func TestApplyResults_UpdatesStats(t *testing.T) {
	id := newMarketID(0x01)
	reader := newMockReader()
	reader.set(id, emptyMarket())

	results := &OnChainResult{
		ID: id,
		Stats: market.MarketStats{
			TotalBorrowAssets: big.NewInt(5000),
			TotalBorrowShares: big.NewInt(4000),
		},
		OraclePrice: big.NewInt(2e18),
	}

	ApplyResults(reader, results)

	m := reader.get(id)
	m.Mu.RLock()
	defer m.Mu.RUnlock()

	if m.Stats.TotalBorrowAssets.Cmp(big.NewInt(5000)) != 0 {
		t.Errorf("TotalBorrowAssets: expected 5000, got %s", m.Stats.TotalBorrowAssets)
	}
	if m.Stats.TotalBorrowShares.Cmp(big.NewInt(4000)) != 0 {
		t.Errorf("TotalBorrowShares: expected 4000, got %s", m.Stats.TotalBorrowShares)
	}
	if m.Oracle.Price.Cmp(big.NewInt(2e18)) != 0 {
		t.Errorf("OraclePrice: expected 2e18, got %s", m.Oracle.Price)
	}
}

func TestApplyResults_NilMarket_NoOp(t *testing.T) {
	id := newMarketID(0x02)
	reader := newMockReader()
	// market non enregistré dans le reader

	results := &OnChainResult{
		ID:          id,
		Stats:       market.MarketStats{TotalBorrowAssets: big.NewInt(1000)},
		OraclePrice: big.NewInt(1e18),
	}

	// ne doit pas paniquer
	ApplyResults(reader, results)
}

func TestApplyResults_OverwritesPreviousValues(t *testing.T) {
	id := newMarketID(0x03)
	reader := newMockReader()

	m := emptyMarket()
	m.Stats.TotalBorrowAssets = big.NewInt(100)
	m.Stats.TotalBorrowShares = big.NewInt(100)
	m.Oracle.Price = big.NewInt(1e18)
	reader.set(id, m)

	results := &OnChainResult{
		ID: id,
		Stats: market.MarketStats{
			TotalBorrowAssets: big.NewInt(9999),
			TotalBorrowShares: big.NewInt(8888),
		},
		OraclePrice: big.NewInt(3e18),
	}

	ApplyResults(reader, results)

	m = reader.get(id)
	m.Mu.RLock()
	defer m.Mu.RUnlock()

	if m.Stats.TotalBorrowAssets.Cmp(big.NewInt(9999)) != 0 {
		t.Errorf("expected 9999, got %s", m.Stats.TotalBorrowAssets)
	}
	if m.Oracle.Price.Cmp(big.NewInt(3e18)) != 0 {
		t.Errorf("expected 3e18, got %s", m.Oracle.Price)
	}
}

// ── OnChainCalls ─────────────────────────────────────────────────────────────

func TestOnChainCalls_ReturnsTwoCalls(t *testing.T) {
	id := newMarketID(0x01)
	reader := newMockReader()
	reader.set(id, emptyMarket())

	mParam := morpho.MarketParams{
		Oracle: common.HexToAddress("0xOracle"),
	}

	calls, _, res := OnChainCalls(reader, mParam, id)

	if len(calls) != 2 {
		t.Errorf("expected 2 RPC calls, got %d", len(calls))
	}
	if res.ID != id {
		t.Errorf("result ID mismatch")
	}
}

func TestOnChainCalls_ResultInitialized(t *testing.T) {
	id := newMarketID(0x01)
	reader := newMockReader()
	reader.set(id, emptyMarket())

	mParam := morpho.MarketParams{
		Oracle: common.HexToAddress("0xOracle"),
	}

	_, _, res := OnChainCalls(reader, mParam, id)

	if res.OraclePrice == nil {
		t.Error("OraclePrice should be initialized")
	}
	if res.ID != id {
		t.Error("ID should be set on result")
	}
}

// ── Concurrency ───────────────────────────────────────────────────────────────

func TestApplyResults_Concurrent_NoRace(t *testing.T) {
	id := newMarketID(0x01)
	reader := newMockReader()
	reader.set(id, emptyMarket())

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int64) {
			defer wg.Done()
			ApplyResults(reader, &OnChainResult{
				ID: id,
				Stats: market.MarketStats{
					TotalBorrowAssets: big.NewInt(n * 1000),
					TotalBorrowShares: big.NewInt(n * 900),
				},
				OraclePrice: big.NewInt(n * 1e18),
			})
		}(int64(i + 1))
	}
	wg.Wait()
}