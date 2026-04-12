package state

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

func MarketReport(marketReader MarketReader, marketMap map[[32]byte]morpho.MarketParams) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, " ---- FOLLOWING %d MARKETS ---- \n", len(marketReader.Ids()))
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

		type riskyPos struct {
			Pos market.BorrowPosition
			hf  *big.Int
		}
		riskyPosArr := []riskyPos{}
		for _, p := range snap.Positions {
			/* sort by hf */

			hf := p.HF(stats.TotalBorrowShares, stats.TotalBorrowAssets, snap.Oracle.Price, snap.LLTV)
			if hf.Cmp(utils.WAD1DOT05) < 0 && hf.Cmp(big.NewInt(0)) > 0 {
				riskyPosArr = append(riskyPosArr, riskyPos{
					Pos: p,
					hf:  hf,
				})
			}

		}
		sort.Slice(riskyPosArr, func(i, j int) bool {
			return riskyPosArr[i].hf.Cmp(riskyPosArr[j].hf) < 0
		})
		if len(riskyPosArr) > 2 {
			for _, r := range riskyPosArr[:2] {
				fmt.Fprintf(&sb, "|  borrower addr:%s hf:%.6f market: \n", r.Pos.Address, utils.BigIntToFloat(r.hf)/1e18)
			}
		} else {
			for _, r := range riskyPosArr {
				fmt.Fprintf(&sb, "|  borrower addr:%s hf:%.6f colassets:%.6f \n", r.Pos.Address, utils.BigIntToFloat(r.hf)/1e18, r.Pos.CollateralAssets)
			}
		}

	}
	return sb.String()

}
