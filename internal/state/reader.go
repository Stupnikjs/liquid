package state

import "github.com/Stupnikjs/morpho-sepolia/internal/cache"

type MarketReader interface {
	Ids() [][32]byte
	GetSnapshot(id [32]byte) *cache.MarketSnapshot
	Update(id [32]byte, fn func(m *cache.Market))
}
