package state

import (
	"math/big"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/position"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/ethereum/go-ethereum/common"
)

type MarketReader interface {
	Ids() [][32]byte
	GetSnapshot(id [32]byte) *market.MarketSnapshot
	Update(id [32]byte, fn func(m *market.Market))
}

// filter out pos with HF > maxHF
func Filter(marketReader MarketReader, maxHF *big.Int) {
	for _, id := range marketReader.Ids() {
		snap := marketReader.GetSnapshot(id)
		if snap == nil {
			continue
		}
		stats := snap.Stats
		if stats.TotalBorrowAssets == nil || stats.TotalBorrowShares == nil || snap.LLTV == nil || snap.Oracle.Price == nil {
			continue
		}
		toKeep := []*position.BorrowPosition{}
		for _, p := range snap.Positions {
			cp := &p
			hf := cp.HF(stats.TotalBorrowShares, stats.TotalBorrowAssets, snap.Oracle.Price, snap.LLTV)
			if hf.Cmp(maxHF) < 0 && hf.Sign() != 0 && hf.Cmp(utils.HALF_WAD) > 0 {
				toKeep = append(toKeep, cp)
			}

		}
		marketReader.Update(id, func(m *market.Market) {
			m.Positions = make(map[common.Address]*position.BorrowPosition, len(toKeep))
			for _, p := range toKeep {
				m.Positions[p.Address] = p
			}

		})
	}
}

func MarketReport(marketReader MarketReader) string {

}
