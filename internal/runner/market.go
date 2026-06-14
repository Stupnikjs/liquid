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
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

const MAX_FLUSH_QUEUE_SIZE = 50 // for test
const NANO_TIME_LIMIT = 2_000_000_000

// One routine per market with dynamic interval based on distance to liquidation (HF = 1)
// Hold Liquidation logic
type marketState struct {
	ignoreMap map[common.Address]int
	tickCount int
	ticker    *time.Ticker
	interval  time.Duration
	fastMode  bool // alchemy on true
}

func (c *MarketConsumer) MarketRoutineWrapper(ctx context.Context, liquidationCh chan cache.BorrowPosition) {
	for _, id := range c.Cache.Ids() {
		go c.MarketRoutine(ctx, liquidationCh, id)
	}
}

func (c *MarketConsumer) MarketRoutine(ctx context.Context, liquidationCh chan cache.BorrowPosition, id [32]byte) {

	ms := &marketState{
		ignoreMap: make(map[common.Address]int),
	}
	ticker, interval := c.MarketInitTicker(ctx, id)
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

			interval := c.MarketTick(ctx, ms, id, liquidationCh)
			if interval != ms.interval {
				ms.ticker.Reset(interval)
			}
			ms.interval = interval
			if interval.Nanoseconds() < NANO_TIME_LIMIT && !ms.fastMode {
				ms.fastMode = true
			}
			if interval.Nanoseconds() >= NANO_TIME_LIMIT && ms.fastMode {
				ms.fastMode = false
			}
		}
	}
}

func (c *MarketConsumer) MarketInitTicker(ctx context.Context, id [32]byte) (*time.Ticker, time.Duration) {
	// Wait for initial data
	var snap *cache.MarketSnapshot
	for snap == nil || snap.Oracle.Price == nil || snap.Oracle.Price.Sign() == 0 {
		select {
		case <-ctx.Done():
			return nil, time.Second
		case <-time.After(500 * time.Millisecond):
			snap = c.Cache.GetSnapshot(id)
		}
	}
	morphoM := c.Cache.MorphoMarketByID(id)
	return SnapToTickerInterval(*snap, morphoM)

}

func SnapToTickerInterval(snap cache.MarketSnapshot, morphoM morpho.MarketParams) (*time.Ticker, time.Duration) {
	firstHF := snap.GetFirstHF()
	if firstHF == nil {
		firstHF = utils.TenPowInt(19)
	}

	diff := utils.DiffWADToFloat(firstHF)
	var interval time.Duration
	if morphoM.IsETHCorrelated() || morphoM.IsBTCCorrelated() {
		diff *= 100
	}
	interval = distanceToInterval(diff)
	return time.NewTicker(interval), interval
}

// OnchainRefresh
func (r *MarketConsumer) MarketTick(ctx context.Context, ms *marketState, id [32]byte, liquidationCh chan cache.BorrowPosition) time.Duration {
	ms.tickCount++

	// Fetch MarketStats OraclePrice
	// Recompute HF and sort
	r.MarketOnchainRefresh(ctx, ms, id)
	r.MarketRecompute(ms, id)

	snap := r.Cache.GetSnapshot(id)
	if snap == nil || len(snap.Positions) == 0 {
		return 10 * time.Hour
	}
	morphoM := r.Cache.MorphoMarketByID(id)
	_, interval := SnapToTickerInterval(*snap, morphoM)

	r.LiquidationCheck(ctx, *snap, ms, liquidationCh)

	err := r.ToDBEntryQueue(id, *snap)
	if err != nil {
		log.Printf("error adding entry to queue: %v", err)
	}
	return interval
}

// RPC Call Oracle only or Markets
func (r *MarketConsumer) MarketOnchainRefresh(ctx context.Context, ms *marketState, id [32]byte) {
	morphoM := r.Cache.MorphoMarketByID(id)
	oracleMethod := *r.Config.MorphoABI.Oracle.Price
	marketMethod := *r.Config.MorphoABI.Blue.Market
	if ms.tickCount%20 == 0 {

		if err := onchain.OnChainRefresh(r.Conn, r.Config.Addresses.Morpho, ctx, r.Cache, morphoM, &oracleMethod, &marketMethod, ms.fastMode); err != nil {
			log.Printf("full refresh error: %v", err)
		}
		return
	}
	if err := onchain.OnChainOracleRefresh(r.Conn, ctx, r.Cache, morphoM, &oracleMethod, ms.fastMode); err != nil {
		log.Printf("oracle refresh error: %v", err)
	}
}

// Sorting + Hf recalculate
func (r *MarketConsumer) MarketRecompute(ms *marketState, id [32]byte) {
	r.Cache.UpdateRecompute(id, ms.tickCount)
}

func (r *MarketConsumer) LiquidationCheck(ctx context.Context, snap cache.MarketSnapshot, ms *marketState, liquidationCh chan cache.BorrowPosition) {
	for _, pos := range snap.Positions {
		// since positions are sorted by HF, we can break early
		if pos.CachedHF == nil || pos.CachedHF.Cmp(utils.WAD) >= 0 {
			break
		}
		// HF under 0.5
		if pos.CachedHF.Cmp(utils.HALF_WAD) < 0 {
			ms.ignoreMap[pos.Address] = 999
			continue
		}
		morphoM := r.Cache.MorphoMarketByID(pos.MarketID)

		// now HF is < 1
		if count, ok := ms.ignoreMap[pos.Address]; !ok || count < 10 {
			log.Printf("sending pos to liquidation chan \n %s \n", pos.String(morphoM, snap.Oracle.Price))
			liquidationCh <- pos
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

func (r *MarketConsumer) ToDBEntryQueue(id [32]byte, snap cache.MarketSnapshot) error {
	r.Store.Mu.Lock()
	defer r.Store.Mu.Unlock()
	for _, pos := range snap.Positions {

		if pos.CachedHF == nil || pos.CachedHF.Cmp(utils.WAD1DOT05) >= 0 {
			break
		}
		e := db.PosToEntry(pos, snap.Oracle.Price, snap.Stats.TotalBorrowAssets, snap.Stats.TotalBorrowShares, time.Now().UnixMilli())
		if len(r.Store.EntryToFlush) < MAX_FLUSH_QUEUE_SIZE {
			r.Store.EntryToFlush = append(r.Store.EntryToFlush, e)
		} else {
			r.Store.FlushEntries()
			r.Store.EntryToFlush = append(r.Store.EntryToFlush, e)
		}

	}
	return nil
}

// EVENTS

func (r *MarketConsumer) SubscribePositionRoutine(ctx context.Context) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{r.Config.Addresses.Morpho},
		Topics: [][]common.Hash{{
			r.Config.MorphoABI.Topics.Borrow,
			r.Config.MorphoABI.Topics.Repay,
			r.Config.MorphoABI.Topics.SupplyCollateral,
			r.Config.MorphoABI.Topics.AccrueInterest,
			r.Config.MorphoABI.Topics.Liquidate,
		}},
	}
	ch, err := r.Conn.SubscribeLogs(ctx, query)
	r.EventCh = ch
	if err != nil {
		log.Printf("Error subscribing to logs: %v", err)
	}

}

func (m *MarketConsumer) EventListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-m.EventCh:
			if !ok {
				return
			}
			onchain.ProcessEvents(m.Cache, event, m.Config.MorphoABI)
		}
	}
}
