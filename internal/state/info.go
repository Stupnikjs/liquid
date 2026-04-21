package state

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/Stupnikjs/morpho-sepolia/internal/cache"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

type MarketInfo struct {
	morpho.MarketParams
	PerctToFirstLiq float64 // 1 - min hf * 100
	Liquidables     []HfPos
	Snap            cache.MarketSnapshot
}

type HfPos struct {
	Pos cache.BorrowPosition
	Hf  *big.Int
}

// this is the only func that calc hf to trigger liquidation
// easier with sorted pos by hf
func CheckMarket(mReader MarketReader, params morpho.MarketParams) MarketInfo {
	info := MarketInfo{}
	info.PerctToFirstLiq = 100
	info.MarketParams = params
	snap := mReader.GetSnapshot(params.ID)
	if snap == nil {
		return info // no data yet, assume safe
	}

	stats := snap.Stats
	var minHf *big.Int

	if len(snap.Positions) == 0 {
		fmt.Println("no pos in snapshot")
		return info
	}
	if snap.Stats.MaxUniSwappable == nil {
		return info
	}
	liquidablesPos := []HfPos{}
	for _, p := range snap.Positions {
		if p.CollateralAssets == nil || p.CollateralAssets.Sign() == 0 {
			continue
		}
		if p.CollateralAssets.Cmp(snap.Stats.MaxUniSwappable) > 0 {
			continue
		}
		hf := p.HF(stats.TotalBorrowShares, stats.TotalBorrowAssets, snap.Oracle.Price, snap.LLTV)
		if hf.Cmp(utils.HALF_WAD) <= 0 {
			continue
		}
		if hf.Cmp(utils.WAD) < 0 {

			liquidablesPos = append(liquidablesPos, HfPos{
				Pos: p,
				Hf:  hf,
			})
		}
		if minHf == nil || hf.Cmp(minHf) < 0 {
			minHf = hf
		}
	}

	// No valid positions found
	if minHf == nil {
		return info
	}

	// Tri par HF croissant — les plus urgentes en premier
	sort.Slice(liquidablesPos, func(i, j int) bool {
		return liquidablesPos[i].Hf.Cmp(liquidablesPos[j].Hf) < 0
	})

	hfFloat := utils.BigIntToFloat(minHf) / 1e18
	info.PerctToFirstLiq = (1 - hfFloat) * 100
	info.Liquidables = liquidablesPos

	info.Liquidables = liquidablesPos

	return info
}
