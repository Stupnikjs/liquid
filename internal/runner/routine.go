package runner

import (
	"context"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
)

func (r *Runner) OnChainRefreshRoutine(ctx context.Context) {
	for _, id := range r.Cache.Markets.Ids() {
		go r.MarketRoutine(ctx, id)
	}

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

	distance := state.GetDistanceFromLiquid(r.Cache.Markets, id)
	interval := distanceToInterval(distance)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			onchain.OnChainRefresh(r.Conn, r.Cache.Markets, r.Cache.GetMorphoMarketFromId(id), id)
			distance = state.GetDistanceFromLiquid(r.Cache.Markets, id)
			newInterval := distanceToInterval(distance)
			if newInterval != interval {
				ticker.Reset(newInterval)
				interval = newInterval
			}
		}
	}
}

func (r *Runner) LogState(ctx context.Context) {
	utils.RunTicker(ctx, 4*time.Minute, func() {
		logs := state.MarketReport(r.Cache.Markets, r.Cache.MarketMap)
		r.Logger <- logs
	})
}

func (r *Runner) ApiCallRoutine(ctx context.Context) error {
	return r.Cache.ApiCall(r.Conn.ClientHTTP, uint32(r.Config.ChainID))
}

func (r *Runner) WatchPositionRoutine(ctx context.Context) {
	r.Conn.WatchPositions(ctx)
}

func (r *Runner) LogEthCallsPerMin(ctx context.Context) {
	r.Conn.LogsEthCallsFromLastMin(ctx, r.Logger)
}
