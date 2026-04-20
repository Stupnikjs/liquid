package state

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
)

/* Need to consume a MarketsInfo struct to output to logs  */
func MarketReport(info MarketInfo) string {
	var sb strings.Builder

	snap := info.Snap
	mParams := info.MarketParams
	stats := snap.Stats

	exposant := 36 + mParams.LoanTokenDecimals - mParams.CollateralTokenDecimals
	bigPrice := new(big.Int).Div(snap.Oracle.Price, utils.TenPowInt(uint(exposant)))
	price := utils.BigIntToFloat(bigPrice)
	borrowAssets := utils.BigIntToFloat(stats.TotalBorrowAssets) / math.Pow10(int(mParams.LoanTokenDecimals))

	fmt.Fprintf(&sb, "═══════════════════════════════════════════\n")
	fmt.Fprintf(&sb, "  %s / %s\n", mParams.CollateralTokenStr, mParams.LoanTokenStr)
	fmt.Fprintf(&sb, "═══════════════════════════════════════════\n")
	fmt.Fprintf(&sb, "  price        %14.6f\n", price)
	fmt.Fprintf(&sb, "  borrow       %14.2f %s\n", borrowAssets, mParams.LoanTokenStr)
	fmt.Fprintf(&sb, "  positions    %14d tracked\n", len(snap.Positions))
	fmt.Fprintf(&sb, "  closest liq  %13.2f%% away\n", info.PerctToFirstLiq)
	fmt.Fprintf(&sb, "  ─────────────────────────────────────────\n")

	if len(info.Liquidables) == 0 {
		fmt.Fprintf(&sb, "  at risk      %14s\n", "none")
	} else {
		fmt.Fprintf(&sb, "  at risk      %14d positions (HF < 1.0)\n", len(info.Liquidables))
		limit := min(2, len(info.Liquidables))
		for _, p := range info.Liquidables[:limit] {
			hf := utils.BigIntToFloat(p.Hf) / 1e18
			fmt.Fprintf(&sb, "  ⚠  %s  HF %.4f  col %.4f\n", p.Pos.Address, hf, p.Pos.CollateralAssets)
		}
	}

	fmt.Fprintf(&sb, "═══════════════════════════════════════════\n")
	return sb.String()
}
