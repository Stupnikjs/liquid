package runner

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/cache"
	market "github.com/Stupnikjs/morpho-sepolia/internal/cache"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/swap"
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
	ticker    *time.Ticker
	interval  time.Duration
}

func (r *Runner) MarketRoutine(ctx context.Context, id [32]byte) {

	ms := &marketState{
		ignoreMap: make(map[common.Address]int),
		tickCount: 0,
	}
	ticker, interval := r.MarketInitTicker(ctx, id)
	if ticker == nil {
		return
	}
	ms.ticker = ticker
	ms.interval = interval
	defer ms.ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ms.ticker.C:
			r.MarketTick(ctx, ms, id)
		}
	}
}

func (r *Runner) MarketInitTicker(ctx context.Context, id [32]byte) (*time.Ticker, time.Duration) {
	// Wait for initial data
	var snap *market.MarketSnapshot
	for snap == nil || snap.Oracle.Price == nil || snap.Oracle.Price.Sign() == 0 {
		select {
		case <-ctx.Done():
			return nil, time.Second
		case <-time.After(500 * time.Millisecond):
			snap = r.Cache.Markets.GetSnapshot(id)
		}
	}
	firstHF := snap.GetFirstHF()
	if firstHF == nil {
		firstHF = utils.TenPowInt(19)
	}
	diff := getDiffFloat(firstHF)
	morphoM := r.Cache.GetMorphoMarketFromId(id)
	var interval time.Duration
	if morphoM.IsETHCorrelated() {
		diff *= 100
	}
	interval = distanceToInterval(diff)
	return time.NewTicker(interval), interval

}

func (r *Runner) MarketTick(ctx context.Context, ms *marketState, id [32]byte) {
	ms.tickCount++
	morphoM := r.Cache.GetMorphoMarketFromId(id)
	start := time.Now().UnixNano()
	err := onchain.OnChainOracleRefresh(r.Conn, ctx, r.Cache.Markets, morphoM, id, r.Config.Addresses.Morpho)
	if err != nil {
		r.Logger <- fmt.Sprintf("Error refreshing on-chain data: %v", err)
	}
	r.Cache.Markets.Update(id, func(m *market.Market) {
		m.RecomputeHFUnsafe(len(m.Positions) / 2)
		if ms.tickCount%20 == 0 {
			err := onchain.OnChainRefresh(r.Conn, ctx, r.Cache.Markets, morphoM, id, r.Config.Addresses.Morpho)
			if err != nil {
				r.Logger <- fmt.Sprintf("Error refreshing on-chain data: %v", err)
			}
			m.RecomputeHFUnsafe(len(m.Positions))
			m.SortAllPositionsByHFUnsafe()

		}
	})
	snap := r.Cache.Markets.GetSnapshot(id)
	r.Logger <- state.GetMarketLog(*snap, id, morphoM)

	firstHF := snap.GetFirstHF()
	if firstHF == nil {
		firstHF = utils.TenPowInt(19)
	}
	diff := getDiffFloat(firstHF)
	if morphoM.IsETHCorrelated() {
		diff *= 100
	}
	if diff < 0 {
		for _, pos := range snap.Positions {
			if pos.CachedHF != nil && pos.CachedHF.Cmp(utils.WAD) < 0 {
				if count, ok := ms.ignoreMap[pos.Address]; !ok || count < 5 {
					r.LiquidateCh <- pos
				}
				ms.ignoreMap[pos.Address]++
			}
		}
	}
	newInterval := distanceToInterval(diff)
	if newInterval != ms.interval {
		ms.ticker.Reset(newInterval)
		ms.interval = newInterval
	}
	end := time.Now().UnixNano()
	fmt.Println("market routine ms:", (end-start)/1e6)
}

func distanceToInterval(distance float64) time.Duration {
	switch {
	// 1% to liquidation or 0.01% for correlated ETH pairs
	case distance < 0.01:
		return 2 * time.Second
	// 2% to liquidation or 0.02% for correlated ETH pairs
	case distance < 0.02:
		return 10 * time.Second
	// 20% to liquidation or 0.2% for correlated ETH pairs
	case distance < 0.20:
		// 8min20sec
		return 500 * time.Second
	default:
		return 800 * time.Second
	}
}

func getDiffFloat(hf *big.Int) float64 {
	diff := new(big.Int).Sub(hf, utils.WAD) // distance to 1
	diffFloat, _ := new(big.Float).SetInt(diff).Float64()
	return diffFloat / 1e18
}

func (r *Runner) Swap(id [32]byte) {
	snap := r.Cache.Markets.GetSnapshot(id)
	if snap == nil {
		return
	}
	morphoM := r.Cache.MarketMap[id]
	result, err := swap.QuoteBinarySearch(r.Conn.ClientHTTP, morphoM, r.Config.Addresses.UniSwapQuoter, snap.Stats.MaxCollateralPos, snap.Oracle.Price)
	if err != nil {
		r.Cache.Markets.Update(id, func(m *cache.Market) {
			m.Canceled = true
		})
		return
	}
	r.Cache.Markets.Update(id, func(m *cache.Market) {
		m.Stats.MaxUniSwappable = result.AmountIn
		m.Stats.SwapFee = result.Fee
	})

}
