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
	fmt.Fprintf(&sb, "═══════════════════════════════════════════\n")
	fmt.Fprintf(&sb, "  MONITORING %d MARKETS\n", len(marketReader.Ids()))
	fmt.Fprintf(&sb, "═══════════════════════════════════════════\n")

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

		fmt.Fprintf(&sb, "\n  %s / %s\n", mParams.CollateralTokenStr, mParams.LoanTokenStr)
		fmt.Fprintf(&sb, "  ─────────────────────────────────────────\n")
		fmt.Fprintf(&sb, "  price        %14.6f\n", price)
		fmt.Fprintf(&sb, "  borrow       %14.2f %s\n", borrowAssets, mParams.LoanTokenStr)
		fmt.Fprintf(&sb, "  positions    %14d tracked\n", len(snap.Positions))

		type riskyPos struct {
			Pos market.BorrowPosition
			hf  *big.Int
		}
		var risky []riskyPos
		for _, p := range snap.Positions {
			hf := p.HF(stats.TotalBorrowShares, stats.TotalBorrowAssets, snap.Oracle.Price, snap.LLTV)
			if hf != nil && hf.Sign() > 0 && hf.Cmp(utils.WAD1DOT05) < 0 && hf.Cmp(utils.HALF_WAD) > 0 {
				risky = append(risky, riskyPos{Pos: p, hf: hf})
			}
		}
		sort.Slice(risky, func(i, j int) bool {
			return risky[i].hf.Cmp(risky[j].hf) < 0
		})

		if len(risky) == 0 {
			fmt.Fprintf(&sb, "  at risk      %14s\n", "none")
		} else {
			fmt.Fprintf(&sb, "  at risk      %14d positions (HF < 1.05)\n", len(risky))
			limit := 2
			if len(risky) < limit {
				limit = len(risky)
			}
			for _, r := range risky[:limit] {
				hf := utils.BigIntToFloat(r.hf) / 1e18
				col := r.Pos.CollateralAssets
				fmt.Fprintf(&sb, "  ⚠  %s  HF %.4f  col %.4f\n", r.Pos.Address, hf, col)
			}
		}
	}

	fmt.Fprintf(&sb, "\n═══════════════════════════════════════════\n")
	return sb.String()
}
