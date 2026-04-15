package state

import (
	"fmt"
	"math/big"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
)

func GetDistanceFromLiquid(mReader MarketReader, id [32]byte) float64 {
	snap := mReader.GetSnapshot(id)
	fmt.Println("snap: ", snap)
	if snap == nil {
		return 1.0 // no data yet, assume safe
	}

	stats := snap.Stats
	var minHf *big.Int

	if len(snap.Positions) == 0 {
		fmt.Println("no pos in snapshot")
		return 1.0
	}
	for _, p := range snap.Positions {
		hf := p.HF(stats.TotalBorrowShares, stats.TotalBorrowAssets, snap.Oracle.Price, snap.LLTV)
		fmt.Println("hf : ", hf)
		// skip zombie positions (likely bad data or dust)
		if hf.Cmp(utils.HALF_WAD) <= 0 {
			continue
		}

		if minHf == nil || hf.Cmp(minHf) < 0 {
			minHf = hf
		}
	}

	// No valid positions found
	if minHf == nil {
		return 1.0
	}

	// Already liquidable
	if minHf.Cmp(utils.WAD) < 0 {
		return 0
	}

	diff, _ := new(big.Int).Sub(minHf, utils.WAD).Float64()
	return diff / 1e18
}
