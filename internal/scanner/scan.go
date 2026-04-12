package scanner

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/engine"
	"github.com/Stupnikjs/morpho-sepolia/internal/logging"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
)

type Runner struct {
	Cache  *Cache
	Engine *engine.LiquidationEngine
	Conn   *connector.Connector
	Logger chan string
	signer *config.Signer
}

func NewRunner(conn *connector.Connector, cache *Cache, signer *config.Signer) *Runner {
	return &Runner{
		Cache:  cache,
		Engine: engine.New(),
		Conn:   conn,
		Logger: logging.NewLogger(context.Background(), "logg"),
		signer: signer,
	}
}

func (r *Runner) Init(ctx context.Context) {
	r.ApiCallRoutine(ctx)
	err := OnChainRefresh(r.Conn, r.Cache)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(len(r.Cache.Markets.Ids()))
	r.FilterMarketBySlippage(ctx)
	fmt.Println(len(r.Cache.Markets.Ids()))
}

func (r *Runner) Run(ctx context.Context) {

	go r.WatchPositionRoutine(ctx)
	// Onchain rpc pool to update markets
	go r.OnChainRefreshRoutine(ctx)
	go r.CleanMarketsRoutine(ctx)
	// Loging Ethcalls per min
	go r.LogEthCallsPerMin(ctx)
	go r.LogState(ctx)
	go r.SimulateCandidatesRoutine(ctx)
	go r.RebuildRoutine(ctx)
	go r.FireLiquidationRoutine(ctx)
	go r.EventLoop(ctx)
	// 👇 bloque proprement
	<-ctx.Done()
}

/* Only into init func no concurencie */
func (r *Runner) FilterMarketBySlippage(ctx context.Context) {
	for _, id := range r.Cache.Markets.Ids() {
		snap := r.Cache.Markets.GetSnapshot(id)
		marketP := r.Cache.marketMap[id]

		if snap == nil || snap.Oracle.Price.Sign() == 0 {
			r.Cache.Markets.Update(id, func(m *market.Market) {
				m.Canceled = true
			}) // pas d'oracle → inutilisable
			continue
		}

		// montant test : 10k$ en unités du collateral
		testAmount := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(marketP.CollateralTokenDecimals)), nil)
		testAmount.Mul(testAmount, big.NewInt(10_000))

		bestFee, bestSlippage := api.FindBestPool(
			r.Conn.ClientHTTP,
			marketP.CollateralToken,
			marketP.LoanToken,
			testAmount,
			snap.Oracle.Price,
		)

		if bestSlippage > 2.0 || bestFee == 0 {
			r.Cache.Markets.Update(id, func(m *market.Market) {
				m.Canceled = true
			})
			continue
		}

		marketP.PoolFee = int32(bestFee)
		r.Cache.marketMap[id] = marketP
	}
}

func (r *Runner) ApiCallRoutine(ctx context.Context) {
	r.Cache.ApiCall(r.Conn.ClientHTTP)
}

func (r *Runner) WatchPositionRoutine(ctx context.Context) {
	r.Conn.WatchPositions(ctx)
}

func (r *Runner) OnChainRefreshRoutine(ctx context.Context) {
	utils.RunTicker(ctx, 2*time.Second, func() {
		OnChainRefresh(r.Conn, r.Cache)
	})
}

func (r *Runner) CleanMarketsRoutine(ctx context.Context) {
	utils.RunTicker(ctx, time.Minute, func() {
		state.Filter(r.Cache.Markets, utils.WAD1DOT1)
	})
}

func (r *Runner) LogState(ctx context.Context) {
	utils.RunTicker(ctx, 4*time.Minute, func() {
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
			simCache := engine.NewSimCache()
			candidates := engine.GetCandidates(r.Cache.Markets, simCache)
			// simulated is sorted by profit
			simulated := engine.SimulateCandidates(r.Conn, r.Cache.Markets, r.Cache.marketMap, candidates, r.Logger, simCache)
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
			liquidateArgs := engine.LiquidateArgs{
				MarketParams: *market.ToMarketContractParams(),
				Borrower:     liquidable.Pos.Address,
				SeizedAssets: liquidable.SeizeAssets,
				RepaidShares: liquidable.RepayShares,
				SwapRouter:   config.BaseUniswapV3Router, // multichain to change
				PoolFee:      big.NewInt(int64(market.PoolFee)),
			}

			engine.LiquidateCall(
				r.signer,
				r.Conn.ClientHTTP,
				ctx,
				liquidateArgs,
			)
		}
	}
}
