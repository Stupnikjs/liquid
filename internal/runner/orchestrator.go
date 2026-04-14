package runner

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/engine"
	"github.com/Stupnikjs/morpho-sepolia/internal/logging"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/swap"
)

type Runner struct {
	Cache  *Cache
	Engine *engine.Engine
	Conn   *connector.Connector
	Logger chan string
	Config config.Config
	// Config avec signer
}

func NewRunner(cache *Cache, conf config.Config) *Runner {
	var logfile string
	if conf.ChainID == 8543 {
		logfile = "base.log"
	} else {
		logfile = "main.log"
	}
	conn := connector.NewConnector(conf.RPC.HTTP, conf.RPC.WS)
	logger := logging.NewLogger(context.Background(), logfile)
	return &Runner{
		Cache:  cache,
		Engine: engine.NewEngine(conn, conf, logger),
		Conn:   conn,
		Logger: logger,
	}
}

func (r *Runner) Init(ctx context.Context) {
	r.ApiCallRoutine(ctx)
	err := onchain.OnChainRefresh(r.Conn, r.Cache.Markets, r.Cache.marketMap, false)
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
	go r.OnChainRefreshRoutineOnlyOracle(ctx)
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

		var testAmount *big.Int
		if strings.Contains(marketP.CollateralTokenStr, "ETH") || strings.Contains(marketP.CollateralTokenStr, "BTC") {
			testAmount = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(marketP.CollateralTokenDecimals)), nil)
			testAmount.Mul(testAmount, big.NewInt(1))

		} else {
			testAmount = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(marketP.CollateralTokenDecimals)), nil)
			testAmount = testAmount.Mul(testAmount, big.NewInt(1000))
			// montant test : 10k$ en unités du collateral
		}

		priceImpact, oracleSlipage := swap.FindBestPool(
			r.Conn.ClientHTTP,
			marketP,
			testAmount,
			snap.Oracle.Price,
		)
		fmt.Println(marketP.CollateralTokenStr, priceImpact)

		if priceImpact > 2.0 || oracleSlipage > 3.0 {
			r.Cache.Markets.Update(id, func(m *market.Market) {
				m.Canceled = true
			})
			continue
		}

	}
}

func (r *Runner) ApiCallRoutine(ctx context.Context) {
	r.Cache.ApiCall(r.Conn.ClientHTTP, uint32(r.Config.ChainID))
}

func (r *Runner) WatchPositionRoutine(ctx context.Context) {
	r.Conn.WatchPositions(ctx)
}


/* Refactor 

Une routine par marché avec event pour refresh call par rapport a distance de la prochaine liquidation 
*/ 
func (r *Runner) OnChainRefreshRoutineOnlyOracle(ctx context.Context) {
	utils.RunTicker(ctx, 2*time.Second, func() {
		onchain.OnChainRefresh(r.Conn, r.Cache.Markets, r.Cache.marketMap, true)
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
			onchain.ProcessEvents(r.Cache.Markets, event)
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
