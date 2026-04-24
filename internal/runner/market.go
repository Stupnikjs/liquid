package runner

import (
	"context"
	"fmt"
	"math/big"
	"time"

	market "github.com/Stupnikjs/morpho-sepolia/internal/cache"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/ethereum/go-ethereum/common"
)

/*
    One routine per market with dynamic interval based on distance to liquidation (HF = 1)
	Hold Liquidation logic

*/

// Dans runner.go — local à la routine, pas dans Market
type marketState struct {
    ignoreMap map[common.Address]int
    tickCount int
}



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
	ignoreMap := make(map[common.Address]int)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			start := time.Now().UnixNano()
			err := onchain.OnChainRefresh(r.Conn, r.Cache.Markets, r.Cache.GetMorphoMarketFromId(id), id, r.Config.Addresses.Morpho)
			if err != nil {
				r.Logger <- fmt.Sprintf("Error refreshing on-chain data: %v", err)
			}
			r.Cache.Markets.Update(id, func(m *market.Market) {
				m.RecomputeHFUnsafe(len(m.Positions))
				m.SortAllPositionsByHFUnsafe() // need to sort less often than recompute
			})
			snap = r.Cache.Markets.GetSnapshot(id)
			firstHF = snap.GetFirstHF()
			if firstHF == nil {
				firstHF = utils.TenPowInt(19)
			}
			diff := getDiffFloat(firstHF)
			if diff < 0 {
				for _, pos := range snap.Positions {
					if pos.CachedHF != nil && pos.CachedHF.Cmp(utils.WAD) < 0 {
						// faire une map[common.Address]int
						// pour compter le nombre de simulation
						// au delà de 20 simulation ignorer
						if count, ok := ignoreMap[pos.Address]; !ok || count < 10 {
							r.LiquidateCh <- pos
						}
						ignoreMap[pos.Address] += 1

					}
				}
			}
			newInterval := distanceToInterval(diff)
			if newInterval != interval {
				ticker.Reset(newInterval)
				interval = newInterval
			}
			end := time.Now().UnixNano()
			fmt.Println("market routine ms:", (end-start)/1e6)
		}
	}
}


func (r *Runner) tryLiquidateActive(id [32]byte, state *marketState) {
    r.Cache.Markets.Update(id, func(m *market.Market) {
        for _, pos := range m.Positions[:m.ActiveIndex] {
            if pos.CachedHF == nil || pos.CachedHF.Cmp(utils.WAD) >= 0 {
                break
            }
            if state.ignoreMap[pos.Address] < 10 {
                r.LiquidateCh <- *pos
            }
            state.ignoreMap[pos.Address]++
        }
    })
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
	case distance < 0.20:
		return 500 * time.Second
	default:
		return 500 * time.Second
	}
}

func getDiffFloat(hf *big.Int) float64 {
	diff := new(big.Int).Sub(hf, utils.WAD) // distance to 1
	diffFloat, _ := new(big.Float).SetInt(diff).Float64()
	return diffFloat / 1e18
}
