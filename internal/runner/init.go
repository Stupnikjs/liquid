package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/internal/connector"
	"github.com/Stupnikjs/liquid/internal/liquidate"
	"github.com/Stupnikjs/liquid/internal/onchain"
	"github.com/Stupnikjs/liquid/internal/swap"
	"github.com/Stupnikjs/liquid/internal/utils"
)

type Runner struct {
	LiquidateConsumer *liquidate.Consumer
	SwapConsumer      *swap.Consumer
	Cache             *cache.Cache
	Conn              *connector.Connector
	Logger            chan string
	Config            config.Config
	LiquidateCh       chan cache.BorrowPosition
}

func NewRunner(initedCache *cache.Cache, conn *connector.Connector, conf config.Config, logfile string) *Runner {
	liquidateCh := make(chan cache.BorrowPosition, 1)
	logger := utils.NewLogger(context.Background(), logfile)
	return &Runner{
		LiquidateConsumer: liquidate.NewConsumer(conn, initedCache.Markets, initedCache.MarketMap, conf, logger, liquidateCh),
		SwapConsumer:      swap.NewConsumer(conn, initedCache.Markets, initedCache.MarketMap, conf, logger),
		Cache:             initedCache,
		Conn:              conn,
		Logger:            logger,
		Config:            conf,
		LiquidateCh:       liquidateCh,
	}
}

func (r *Runner) Init(ctx context.Context) {
	err := r.ApiCall(ctx)
	if err != nil {
		r.Logger <- err.Error()
	}
	r.Logger <- "Api call init"
	r.OnChainRefreshAll(ctx)
	r.Logger <- "Refresh all markets for init"
	r.FindSwapRoutes()
	r.Logger <- "Quoting over "
	r.LogMarkets()

}

func (r *Runner) LogMarkets() {
	ids := r.Cache.Markets.Ids()
	for _, id := range ids {
		m := r.Cache.GetMorphoMarketFromId(id)
		snap := r.Cache.Markets.GetSnapshot(id)
		if snap == nil {
			continue
		}
		r.log(fmt.Sprintf("[%s/%s]",
			m.CollateralTokenStr,
			m.LoanTokenStr))
	}
}

// on chain call for all active market
func (r *Runner) OnChainRefreshAll(ctx context.Context) {
	var wg sync.WaitGroup
	for _, id := range r.Cache.Markets.Ids() {
		wg.Add(1)
		go func(id [32]byte) {
			defer wg.Done()
			onchain.OnChainRefresh(r.Conn, ctx, r.Cache.Markets, r.Cache.GetMorphoMarketFromId(id), id, r.Config.Addresses.Morpho)
		}(id)
	}

	wg.Wait()

}
