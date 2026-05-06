package runner

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	market "github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/onchain"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/swap"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
			interval := r.MarketTick(ctx, ms, id)
			if interval != ms.interval {
				ms.ticker.Reset(interval)
			}
			ms.interval = interval
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
	return r.SnapToTickerInterval(*snap, id)

}

func (r *Runner) SnapToTickerInterval(snap cache.MarketSnapshot, id [32]byte) (*time.Ticker, time.Duration) {
	firstHF := snap.GetFirstHF()
	if firstHF == nil {
		firstHF = utils.TenPowInt(19)
	}
	morphoM := r.Cache.GetMorphoMarketFromId(id)
	r.log(fmt.Sprintf("%s oracle_price:%s  hf:%s", morphoM.GetPair(), snap.Oracle.Price.String(), utils.FormatWAD(firstHF)))
	diff := getDiffFloat(firstHF)

	var interval time.Duration
	if morphoM.IsETHCorrelated() {
		diff *= 100
	}
	interval = distanceToInterval(diff)
	return time.NewTicker(interval), interval
}

func (r *Runner) MarketTick(ctx context.Context, ms *marketState, id [32]byte) time.Duration {
	ms.tickCount++

	r.MarketOnchainRefresh(ctx, ms, id)
	r.MarketRecompute(ms, id)
	snap := r.Cache.Markets.GetSnapshot(id)
	if snap == nil || len(snap.Positions) == 0 {
		r.log("snap is nil or has no pos")
		return 10 * time.Hour
	}
	_, interval := r.SnapToTickerInterval(*snap, id)
	r.LiquidationCheck(ctx, *snap, ms)
	return interval
}

// RPC Call Oracle only or Markets
func (r *Runner) MarketOnchainRefresh(ctx context.Context, ms *marketState, id [32]byte) {
	morphoM := r.Cache.GetMorphoMarketFromId(id)
	if ms.tickCount%20 == 0 {

		if err := onchain.OnChainRefresh(r.Conn, ctx, r.Cache.Markets, morphoM, id, r.Config.Addresses.Morpho); err != nil {
			r.log(fmt.Sprintf("full refresh error: %v", err))
		}
		return
	}
	if err := onchain.OnChainOracleRefresh(r.Conn, ctx, r.Cache.Markets, morphoM, id, r.Config.Addresses.Morpho); err != nil {
		r.log(fmt.Sprintf("oracle refresh error: %v", err))
	}
}

// Sorting + Hf recalculate
func (r *Runner) MarketRecompute(ms *marketState, id [32]byte) {
	r.Cache.Markets.Update(id, func(m *market.Market) {
		m.RecomputeHFUnsafe(len(m.Positions) / 2)
		if ms.tickCount%10 == 0 {
			m.RecomputeHFUnsafe(len(m.Positions))
			m.SortAllPositionsByHFUnsafe()

		}
	})

}

func (r *Runner) LiquidationCheck(ctx context.Context, snap cache.MarketSnapshot, ms *marketState) {
	for _, pos := range snap.Positions {
		if pos.CachedHF == nil || pos.CachedHF.Cmp(utils.WAD) >= 0 {
			break
		}
		if pos.CachedHF.Cmp(utils.HALF_WAD) < 0 {
			ms.ignoreMap[pos.Address] = 999
			continue
		}
		morphoM := r.Cache.GetMorphoMarketFromId(snap.ID)
		if count, ok := ms.ignoreMap[pos.Address]; !ok || count < 5 {
			r.log(fmt.Sprintf("liquidation check borrower %s usd:%s for market %s %s hf:%s collateralAsset:%s oraclePrice:%s",
				pos.Address,
				utils.FormatWAD(pos.BorrowAssetsUsd),
				morphoM.GetPair(),
				hexutil.Encode(pos.MarketID[:]),
				utils.FormatWAD(pos.CachedHF),
				utils.FormatDecimals(pos.CollateralAssets, int(morphoM.CollateralTokenDecimals)),
				snap.Oracle.Price.String()))
			r.LiquidateCh <- pos
		}
		ms.ignoreMap[pos.Address]++
	}
}

func distanceToInterval(distance float64) time.Duration {
	switch {
	case distance < 0.002:
		return 1 * time.Second
	case distance < 0.005:
		return 2 * time.Second
	// 2% to liquidation or 0.02% for correlated ETH pairs
	case distance < 0.01:
		return 4 * time.Second
	case distance < 0.02:
		return 10 * time.Second
	// 20% to liquidation or 0.2% for correlated ETH pairs
	case distance < 0.10:
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
