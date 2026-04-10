package state

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
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

		toKeep := []*market.BorrowPosition{}
		for _, p := range snap.Positions {
			cp := &p

			hf := cp.HF(stats.TotalBorrowShares, stats.TotalBorrowAssets, snap.Oracle.Price, snap.LLTV)
			if hf.Cmp(maxHF) < 0 && hf.Sign() != 0 && hf.Cmp(utils.HALF_WAD) > 0 {
				toKeep = append(toKeep, cp)
			}

		}
		if len(toKeep) < 2 {
			marketReader.Update(id, func(m *market.Market) {
				m.Canceled = true
			})
			continue
		}
		marketReader.Update(id, func(m *market.Market) {
			m.Positions = make(map[common.Address]*market.BorrowPosition, len(toKeep))
			for _, p := range toKeep {
				m.Positions[p.Address] = p
			}

		})
	}

}

func MarketReport(marketReader MarketReader, marketMap map[[32]byte]morpho.MarketParams) string {
	var sb strings.Builder
	for _, id := range marketReader.Ids() {
		snap := marketReader.GetSnapshot(id)
		if snap == nil {
			continue
		}
		mParams := marketMap[id]
		stats := snap.Stats

		if stats.TotalBorrowAssets == nil || stats.TotalBorrowShares == nil || snap.LLTV == nil || snap.Oracle.Price == nil {
			continue
		}
		exposant := 36 + mParams.LoanTokenDecimals - mParams.CollateralTokenDecimals
		price := utils.BigIntToFloat(snap.Oracle.Price) / math.Pow10(int(exposant))
		borrowAssets := utils.BigIntToFloat(stats.TotalBorrowAssets) / math.Pow10(int(mParams.LoanTokenDecimals))
		borrowShares := utils.BigIntWADToFloat(stats.TotalBorrowShares)
		fmt.Fprintf(&sb, "\n┌─ Market %s/%s\n", mParams.CollateralTokenStr, mParams.LoanTokenStr)
		fmt.Fprintf(&sb, "│  price:         %.6f\n", price)
		fmt.Fprintf(&sb, "│  borrow assets: %.2f\n", borrowAssets)
		fmt.Fprintf(&sb, "│  borrow shares: %.2f\n", borrowShares)
		fmt.Fprintf(&sb, "│  positions less than 10pct from liquidation: %d\n", len(snap.Positions))

		for _, p := range snap.Positions {

			hf := p.HF(stats.TotalBorrowShares, stats.TotalBorrowAssets, snap.Oracle.Price, snap.LLTV)
			if hf.Cmp(utils.WAD1DOT01) < 1 {
				// refactor
				fmt.Fprintf(&sb, "│  hf %f : %s  %f  %s\n", utils.BigIntToFloat(hf)/1e18, p.Address, utils.BigIntToFloat(snap.Oracle.Price)/1e36, mParams.CollateralTokenStr)
			}

		}
	}
	return sb.String()

}
