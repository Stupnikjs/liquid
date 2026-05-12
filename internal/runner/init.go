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
	Store             lqtypes.Store
	SwapRoutes        *swap.RouteCache
	Infra             *lqtypes.Infra
	Logger            chan string
	LiquidateCh       chan cache.BorrowPosition
}

func NewRunner(infra *lqtypes.Infra, store lqtypes.Store, logfile string) *Runner {
	liquidateCh := make(chan cache.BorrowPosition, 1)
	logger := utils.NewLogger(context.Background(), logfile)
	return &Runner{
		LiquidateConsumer: liquidate.NewConsumer(infra, store, logger, liquidateCh),
		Store:             store,
		SwapRoutes: &swap.RouteCache{
			Routes: make(map[swap.RouteKey][]lqtypes.PoolEdge, len(store.MarketMap)),
		},
		Infra:       infra,
		Logger:      logger,
		LiquidateCh: liquidateCh,
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
	r.SingleHop()
	r.Logger <- "Quoting over "
	r.LogMarkets()

}

func (r *Runner) LogMarkets() {
	ids := r.Store.MarketReader.Ids()
	for _, id := range ids {
		m := r.Store.MarketMap[id]
		snap := r.Store.MarketReader.GetSnapshot(id)
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
	for _, id := range r.Store.MarketReader.Ids() {
		wg.Add(1)
		go func(id [32]byte) {
			m := r.Store.MarketMap[id]
			defer wg.Done()
			onchain.OnChainRefresh(r.Infra, ctx, r.Store.MarketReader, m, id)
		}(id)
	}

	wg.Wait()

}
