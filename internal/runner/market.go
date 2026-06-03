package runner

import (
	"context"
	"log"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/db"
	"github.com/Stupnikjs/liquid/internal/onchain"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var MAX_FLUSH_QUEUE_SIZE = 5_000

// One routine per market with dynamic interval based on distance to liquidation (HF = 1)
// Hold Liquidation logic
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
		log.Println("Failed to initialize ticker")
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
	var snap *cache.MarketSnapshot
	for snap == nil || snap.Oracle.Price == nil || snap.Oracle.Price.Sign() == 0 {
		select {
		case <-ctx.Done():
			return nil, time.Second
		case <-time.After(500 * time.Millisecond):
			snap = r.MarketConsumer.Cache.Markets.GetSnapshot(id)
		}
	}
	morphoM := r.MarketConsumer.Cache.MarketMap[id]
	return r.SnapToTickerInterval(*snap, morphoM)

}

func (r *Runner) SnapToTickerInterval(snap cache.MarketSnapshot, morphoM morpho.MarketParams) (*time.Ticker, time.Duration) {
	firstHF := snap.GetFirstHF()
	if firstHF == nil {
		firstHF = utils.TenPowInt(19)
	}

	diff := utils.DiffWADToFloat(firstHF)
	var interval time.Duration
	if morphoM.IsETHCorrelated() {
		diff *= 100
	}
	interval = distanceToInterval(diff)
	return time.NewTicker(interval), interval
}

func (r *Runner) MarketTick(ctx context.Context, ms *marketState, id [32]byte) time.Duration {
	start := time.Now().UnixNano()
	ms.tickCount++
	r.MarketOnchainRefresh(ctx, ms, id)
	r.MarketRecompute(ms, id)
	snap := r.MarketConsumer.Cache.Markets.GetSnapshot(id)
	if snap == nil || len(snap.Positions) == 0 {
		log.Println("snap is nil or has no pos")
		return 10 * time.Hour
	}

	morphoM := r.MarketConsumer.Cache.MarketMap[id]

	_, interval := r.SnapToTickerInterval(*snap, morphoM)
	r.LiquidationCheck(ctx, *snap, ms)
	latency := (time.Now().UnixNano() - start) / 1_000_000
	if ms.tickCount%100 == 0 {
		log.Printf("latency:%d %s oracle_price:%s  hf:%f", latency, morphoM.GetPair(), snap.Oracle.Price.String(), utils.BigIntWADToFloat(snap.GetFirstHF()))
	}
	log.Printf("%s %s", morphoM.GetPair(), snap.Analysis().String())
	err := r.ToDBEntryQueue(id, *snap)
	if err != nil {
		log.Printf("error adding entry to queue: %v", err)
	}
	return interval
}

// RPC Call Oracle only or Markets
func (r *MarketConsumer) MarketOnchainRefresh(ctx context.Context, ms *marketState, id [32]byte) {
	morphoM := r.Cache.MarketMap[id]
	if ms.tickCount%20 == 0 {

		if err := onchain.OnChainRefresh(r.Conn, r.Config, ctx, r.Cache, morphoM, id); err != nil {
			log.Printf("full refresh error: %v", err)
		}
		return
	}
	if err := onchain.OnChainOracleRefresh(r.Conn, r.Config, ctx, r.Cache, morphoM, id, r.Config.Addresses.Morpho); err != nil {
		log.Printf("oracle refresh error: %v", err)
	}
}

// Sorting + Hf recalculate
func (r *MarketConsumer) MarketRecompute(ms *marketState, id [32]byte) {
	r.Cache.Markets.Update(id, func(m *cache.Market) {
		m.RecomputeHFUnsafe(len(m.Positions) / 2)
		if ms.tickCount%10 == 0 {
			m.RecomputeHFUnsafe(len(m.Positions))
			m.SortAllPositionsByHFUnsafe()

		}
	})

}

// Checking hf and sending pos into liquidation channel
// updating market state ignore map on malformed pos
func (r *Runner) LiquidationCheck(ctx context.Context, snap cache.MarketSnapshot, ms *marketState) {
	for _, pos := range snap.Positions {
		// since positions are sorted by HF, we can break early
		if pos.CachedHF == nil || pos.CachedHF.Cmp(utils.WAD) >= 0 {
			break
		}
		// baddebt
		if pos.CachedHF.Cmp(utils.HALF_WAD) < 0 {
			ms.ignoreMap[pos.Address] = 999
			continue
		}
		morphoM := r.MarketConsumer.Cache.MarketMap[snap.ID]
		if count, ok := ms.ignoreMap[pos.Address]; !ok || count < 10 {
			log.Printf("liquidation check borrower %s usd:%s for market %s %s hf:%s collateralAsset:%s oraclePrice:%s",
				pos.Address,
				utils.FormatWAD(pos.BorrowAssetsUsd),
				morphoM.GetPair(),
				hexutil.Encode(pos.MarketID[:]),
				utils.FormatWAD(pos.CachedHF),
				utils.FormatDecimals(pos.CollateralAssets, int(morphoM.CollateralTokenDecimals)),
				snap.Oracle.Price.String())
			r.LiquidateConsumer.Ch <- pos
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

// query quoter to test swaping
// updating market if swapable
// canceling if not

func (r *Runner) ToDBEntryQueue(id [32]byte, snap cache.MarketSnapshot) error {
	for _, pos := range snap.Positions {
		if pos.CachedHF == nil || pos.CachedHF.Cmp(utils.WAD1DOT05) >= 0 {
			break // slice trié, on peut break
		}
		e := db.PosToEntry(pos, snap.Oracle.Price, snap.Stats.TotalBorrowAssets, snap.Stats.TotalBorrowShares, time.Now().UnixMilli())
		if len(r.MarketConsumer.Store.EntryToFlush) < MAX_FLUSH_QUEUE_SIZE {
			r.MarketConsumer.Store.EntryToFlush = append(r.MarketConsumer.Store.EntryToFlush, e)
		} else {
			r.FlushEntries()
			r.MarketConsumer.Store.EntryToFlush = append(r.MarketConsumer.Store.EntryToFlush, e)
		}

	}
	return nil
}

// need mutex here
func (r *Runner) FlushEntries() {
	if len(r.MarketConsumer.Store.EntryToFlush) == 0 {
		return
	}
	err := db.InsertEntries(r.MarketConsumer.Store.DB, r.MarketConsumer.Store.EntryToFlush)
	if err != nil {
		log.Printf("error flushing entries: %v", err)
	}
	r.MarketConsumer.Store.EntryToFlush = r.MarketConsumer.Store.EntryToFlush[:0] // Clear the slice
}
