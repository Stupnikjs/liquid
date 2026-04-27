package state

import (
	"fmt"
	"strings"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

// just log market pair and first pos
func GetMarketLog(c MarketReader, id [32]byte, morphoM morpho.MarketParams) string {
	snap := c.GetSnapshot(id)

	var sb strings.Builder

	marketPair := fmt.Sprintf("%s/%s ", morphoM.CollateralTokenStr, morphoM.LoanTokenStr)

	if snap == nil {
		fmt.Fprintf(&sb, "%s", marketPair)
		sb.WriteString("(empty snapshot)\n")
		return sb.String()
	}

	fmt.Fprintf(&sb, "%s %d pos", marketPair, len(snap.Positions))

	priceStr := "nil"
	if snap.Oracle.Price != nil {
		priceStr = utils.FormatDecimals(
			snap.Oracle.Price,
			int(36+morphoM.LoanTokenDecimals-morphoM.CollateralTokenDecimals),
		)
	}

	limit := min(len(snap.Positions), 1)

	for i := range limit {
		pos := snap.Positions[i]

		fmt.Fprintf(&sb,
			"BorrowShares=%s, Collateral=%s, Oracle=%s  , HF=%s\n",
			utils.FormatDecimals(pos.BorrowShares, 18),
			utils.FormatDecimals(pos.CollateralAssets, int(morphoM.CollateralTokenDecimals)),
			priceStr,
			utils.FormatDecimals(pos.CachedHF, 18),
		)
	}
	return sb.String()
}
