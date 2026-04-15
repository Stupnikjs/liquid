package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/engine"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
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
			onchain.OnChainRefresh(r.Conn, r.Cache.Markets, r.Cache.marketMap[id], id)
			fmt.Println(len(r.Cache.Markets.Ids()))
			distance = state.GetDistanceFromLiquid(r.Cache.Markets, id)
			fmt.Println("dist: ", distance)
			fmt.Println("oracle: ", snap.Oracle.Price)
			newInterval := distanceToInterval(distance)
			if newInterval != interval {
				ticker.Reset(newInterval)
				interval = newInterval
			}
		}
	}
}

/*
func (r *Runner) CleanMarketsRoutine(ctx context.Context) {
	utils.RunTicker(ctx, time.Minute, func() {
		state.Filter(r.Cache.Markets, utils.WAD1DOT1)
	})
}
*/

func (r *Runner) LogState(ctx context.Context) {
	utils.RunTicker(ctx, 4*time.Minute, func() {
		logs := state.MarketReport(r.Cache.Markets, r.Cache.marketMap)
		r.Logger <- logs
	})
}

func (r *Runner) ApiCallRoutine(ctx context.Context) {
	api.ApiCall(r.Conn.ClientHTTP, r.Cache.Markets, r.Cache.marketMap, uint32(r.Config.ChainID))
}

func (r *Runner) WatchPositionRoutine(ctx context.Context) {
	r.Conn.WatchPositions(ctx)
}

func (r *Runner) RebuildRoutine(ctx context.Context) {
	utils.RunTicker(ctx, 4*time.Second, func() {
		r.Engine.RebuildCh <- true
	})
}

func (r *Runner) LogEthCallsPerMin(ctx context.Context) {
	r.Conn.LogsEthCallsFromLastMin(ctx, r.Logger)
}

func (r *Runner) SimulateCandidatesRoutine(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-r.Engine.RebuildCh:
			_ = event
			if !ok {
				return
			}
			simCache := engine.NewSimCache()
			candidates := engine.GetCandidates(r.Cache.Markets, simCache)
			simulated := engine.SimulateCandidates(r.Conn, r.Cache.Markets, r.Cache.marketMap, candidates, r.Logger, simCache)
			for _, l := range simulated {
				if l.IsLiquidable {
					r.Engine.LiquidateCh <- l
				}
			}
		}

	}

}

func (r *Runner) FireLiquidationRoutine(ctx context.Context) {
	for {
		select {

		case <-ctx.Done():
			return
		case liquidable := <-r.Engine.LiquidateCh:
			market := r.Cache.GetMorphoMarketFromId(liquidable.MarketID)

			liquidateArgs := engine.LiquidateArgs{
				MarketParams: *market.ToMarketContractParams(),
				Borrower:     liquidable.Pos.Address,
				SeizedAssets: liquidable.SeizeAssets,
				RepaidShares: liquidable.RepayShares,
				OdosRouter:   config.OdosRouterAddr,
				OdosCallData: liquidable.OdosCallData,
			}

			r.Engine.ExecuteLiquidation(
				ctx,
				liquidable,
				liquidateArgs,
			)
		}
	}
}
