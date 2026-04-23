package state

import (
	"fmt"
	"strings"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

func GetMarketLog(c MarketReader, id [32]byte, morphoM morpho.MarketParams) string {
	snap := c.GetSnapshot(id)

	var sb strings.Builder

	title := fmt.Sprintf("\n ________________  Market %s/%s _________________ ", morphoM.CollateralTokenStr, morphoM.LoanTokenStr)

	if snap == nil {
		fmt.Fprintf(&sb, "%s\n", title)
		sb.WriteString(" (empty snapshot)\n")
		fmt.Fprintf(&sb, "%s\n", title)
		return sb.String()
	}

	fmt.Fprintf(&sb, "%s\n", title)

	priceStr := "nil"
	if snap.Oracle.Price != nil {
		priceStr = utils.FormatDecimals(
			snap.Oracle.Price,
			int(36+morphoM.LoanTokenDecimals-morphoM.CollateralTokenDecimals),
		)
	}

	limit := min(len(snap.Positions), 6)

	for i := 0; i < limit; i++ {
		pos := snap.Positions[i]

		fmt.Fprintf(&sb,
			" Pos %d: BorrowShares=%s, Collateral=%s, Oracle=%s  , HF=%s\n",
			i,
			utils.FormatDecimals(pos.BorrowShares, 18),
			utils.FormatDecimals(pos.CollateralAssets, int(morphoM.CollateralTokenDecimals)),
			priceStr,
			utils.FormatDecimals(pos.CachedHF, 18),
		)
	}

	return sb.String()
}
