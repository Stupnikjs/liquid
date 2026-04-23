package runner

import (
	"context"
	"math/big"
	"time"

	market "github.com/Stupnikjs/morpho-sepolia/internal/cache"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
)

func (r *Runner) MarketRoutine(ctx context.Context, id [32]byte) {
	// Wait for initial data
	var snap *market.MarketSnapshot
	for snap == nil || snap.Oracle.Price == nil || snap.Oracle.Price.Sign() == 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
			snap = r.Cache.Markets.GetSnapshot(id)
		}
	}
	firstHF := snap.GetFirstHF()
	if firstHF == nil {
		firstHF = utils.TenPowInt(19)
	}
	diff := getDiffFloat(firstHF)
	interval := distanceToInterval(diff)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			onchain.OnChainRefresh(r.Conn, r.Cache.Markets, r.Cache.GetMorphoMarketFromId(id), id, r.Config.Addresses.Morpho)
			r.Cache.Markets.Update(id, func(m *market.Market) {
				m.RecomputeActiveHF()
			})
			snap = r.Cache.Markets.GetSnapshot(id)
			snap.Log(r.Cache.GetMorphoMarketFromId(id))
			firstHF = snap.GetFirstHF()
			if firstHF == nil {
				firstHF = utils.TenPowInt(19)
			}
			diff := getDiffFloat(firstHF)
			if diff < 0 {
				for _, pos := range snap.Positions {
					if pos.CachedHF != nil && pos.CachedHF.Cmp(utils.WAD) < 0 {
						r.LiquidateCh <- pos
					}
				}
			}
			newInterval := distanceToInterval(diff)
			if newInterval != interval {
				ticker.Reset(newInterval)
				interval = newInterval
			}
		}
	}
}

func distanceToInterval(distance float64) time.Duration {
	switch {
	// 1% if ETH pair < 0.0001
	case distance < 0.01:
		return 2 * time.Second
	// 1% if ETH pair < 0.0003
	case distance < 0.03:
		return 10 * time.Second
	// 1% if ETH pair < 0.0005
	case distance < 0.05:
		return 100 * time.Second
	default:
		return 200 * time.Second
	}
}

func getDiffFloat(hf *big.Int) float64 {
	diff := new(big.Int).Sub(hf, utils.WAD) // distance to 1
	diffFloat, _ := new(big.Float).SetInt(diff).Float64()
	return diffFloat / 1e18
}
