package testutil

import (
	"context"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/pkg/morpho"
)

type MockCache struct {
	IdsVal      [][32]byte
	SnapshotMap map[[32]byte]*cache.MarketSnapshot
	MarketMap   map[[32]byte]morpho.MarketParams
	ApiCallErr  error
}

func (m *MockCache) ApiCall() error                                { return m.ApiCallErr }
func (m *MockCache) ApiResyncRoutine(ctx context.Context)          {}
func (m *MockCache) Ids() [][32]byte                               { return m.IdsVal }
func (m *MockCache) GetSnapshot(id [32]byte) *cache.MarketSnapshot { return m.SnapshotMap[id] }
func (m *MockCache) MarketByID(id [32]byte) morpho.MarketParams    { return m.MarketMap[id] }
func (m *MockCache) Update(id [32]byte, fn func(*cache.Market)) {
	// optionnel : appliquer fn sur une copie locale pour tester les mutations
}
func (m *MockCache) LogMarkets() {}
