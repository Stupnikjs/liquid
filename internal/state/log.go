package state

import (
	"fmt"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
)

type MarketLog struct {
	MarketID    string
	OraclePrice string
	TotalBorrow string
	Positions   int
	FirstHF     string
	Liquidables int
	Interval    string
	Timestamp   string
}

func GetMarketLog(c MarketReader, id [32]byte, liquidables int, interval time.Duration, borrowDecimals int) MarketLog {
	snap := c.GetSnapshot(id)
	return MarketLog{
		MarketID:    fmt.Sprintf("%x", id[:4]),
		OraclePrice: utils.FormatWAD(snap.Oracle.Price),
		TotalBorrow: utils.FormatDecimals(snap.Stats.TotalBorrowAssets, borrowDecimals),
		Positions:   len(snap.Positions),
		FirstHF: func() string {
			if len(snap.Positions) == 0 {
				return "none"
			}
			return utils.FormatWAD(snap.Positions[0].CachedHF)
		}(),
		Liquidables: liquidables,
		Interval:    interval.Round(time.Millisecond).String(),
		Timestamp:   time.Now().Format(time.DateTime),
	}
}
