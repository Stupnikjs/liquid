package state

import (
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
)

type MarketReader interface {
	Ids() [][32]byte
	GetSnapshot(id [32]byte) *market.MarketSnapshot
	Update(id [32]byte, fn func(m *market.Market))
}
