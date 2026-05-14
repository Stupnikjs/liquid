package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/liquidate"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/internal/onchain"
	"github.com/Stupnikjs/liquid/internal/swap"
	"github.com/Stupnikjs/liquid/internal/utils"
)

type Runner struct {
	LiquidateConsumer *liquidate.Consumer
	SwapConsumer      *swap.Consumer
	Cache             *cache.Cache
	SwapRoutes        *swap.RouteCache
	Infra             *lqtypes.Infra
	Logger            chan string
	LiquidateCh       chan cache.BorrowPosition
}

func NewRunner(infra *lqtypes.Infra, routeCache *swap.RouteCache, mCache *cache.Cache, logfile string) *Runner {
	liquidateCh := make(chan cache.BorrowPosition, 1)
	logger := utils.NewLogger(context.Background(), logfile)
	return &Runner{
		LiquidateConsumer: liquidate.NewConsumer(infra, mCache, routeCache, logger, liquidateCh),
		SwapConsumer:      swap.NewConsumer(infra, mCache, routeCache, logger),
		Cache:             mCache,
		SwapRoutes:        routeCache,
		Infra:             infra,
		Logger:            logger,
		LiquidateCh:       liquidateCh,
	}
}

func (r *Runner) Init(ctx context.Context) {
	err := r.ApiCall()
	if err != nil {
		r.Logger <- err.Error()
	}
	r.Logger <- "Api call init"
	r.OnChainRefreshAll(ctx)
	r.Logger <- "Refresh all markets for init"
	r.SwapConsumer.RouteCacheRefresh()
	r.Logger <- "Quoting over "
	r.LogMarkets()

}

func (r *Runner) LogMarkets() {
	ids := r.Cache.Markets.Ids()
	for _, id := range ids {
		m := r.Cache.MarketMap[id]
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
			m := r.Cache.MarketMap[id]
			defer wg.Done()
			err := onchain.OnChainRefresh(r.Infra, ctx, r.Cache, m, id)
			if err != nil {
				fmt.Println(err)
			}

		}(id)
	}

	wg.Wait()

}
