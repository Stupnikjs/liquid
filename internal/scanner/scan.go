package scanner

import (
	"context"
	"math/big"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/engine"
	"github.com/Stupnikjs/morpho-sepolia/internal/logging"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

type Runner struct {
	Cache  *Cache
	Engine *engine.LiquidationEngine
	Conn   *connector.Connector
	Logger chan string
	signer *morpho.Signer
}

func NewRunner(conn *connector.Connector, cache *Cache, signer *morpho.Signer) *Runner {
	return &Runner{
		Cache:  cache,
		Engine: engine.New(),
		Conn:   conn,
		Logger: logging.NewLogger(context.Background(), "logg"),
		signer: signer,
	}
}

func (r *Runner) Run(ctx context.Context) {
	r.ApiCallRoutine(ctx)
	go r.WatchPositionRoutine(ctx)
	// Onchain rpc pool to update markets
	go r.OnChainRefreshRoutine(ctx)
	go r.CleanMarketsRoutine(ctx)
	// Loging Ethcalls per min
	go r.LogEthCallsPerMin(ctx)
	// go r.LogState(ctx)
	go r.SimulateCandidatesRoutine(ctx)
	go r.RebuildRoutine(ctx)
	go r.FireLiquidationRoutine(ctx)
	go r.EventLoop(ctx)
	// go r.PrintSlippage(ctx)
	// 👇 bloque proprement
	<-ctx.Done()
}

func (r *Runner) ApiCallRoutine(ctx context.Context) {
	r.Cache.ApiCall(r.Conn.ClientHTTP)
}

func (r *Runner) WatchPositionRoutine(ctx context.Context) {
	r.Conn.WatchPositions(ctx)
}

func (r *Runner) OnChainRefreshRoutine(ctx context.Context) {
	utils.RunTicker(ctx, 5*time.Second, func() {
		OnChainRefresh(r.Conn, r.Cache)
	})
}

func (r *Runner) CleanMarketsRoutine(ctx context.Context) {
	utils.RunTicker(ctx, time.Minute, func() {
		state.Filter(r.Cache.Markets, utils.WAD1DOT1)
	})
}

func (r *Runner) LogState(ctx context.Context) {
	utils.RunTicker(ctx, 10*time.Second, func() {
		logs := state.MarketReport(r.Cache.Markets, r.Cache.marketMap)
		r.Logger <- logs
	})
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
			candidates := engine.GetCandidates(r.Cache.Markets)
			// simulated is sorted by profit
			simulated := engine.SimulateCandidates(r.Conn, r.Cache.Markets, r.Cache.marketMap, candidates, r.Logger)
			for _, l := range simulated {
				if l.IsLiquidable {
					r.Engine.LiquidateCh <- l
				}
			}
		}

	}

}

func (r *Runner) EventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-r.Conn.PositionCh:
			if !ok {
				return
			}
			r.Cache.ProcessEvents(event)
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
			engine.LiquidateCall(
				r.signer,
				r.Cache.Markets,
				r.Conn.ClientHTTP,
				ctx,
				*market.ToMarketContractParams(),
				liquidable.Pos.Address,
				liquidable.SeizeAssets,
				liquidable.RepayShares,
				config.BaseUniswapV3Router, // multichain to change
				big.NewInt(int64(market.PoolFee)))
		}
	}
}
