type MarketReader interface {
	Ids() [][32]byte
	GetSnapshot(id [32]byte) *MarketSnapshot
	Update(id [32]byte, fn func(m *Market))
}