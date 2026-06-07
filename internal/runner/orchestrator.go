package runner

import (
	"context"
	"fmt"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/pkg/api"
	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/Stupnikjs/liquid/pkg/swap"
)

// separér les dependances par routine

func (r *Runner) Run(ctx context.Context) {
	go r.MarketConsumer.SubscribePositionRoutine(ctx)
	go r.MarketConsumer.OnChainRefreshRoutine(ctx, r.LiquidateConsumer.Ch)
	go r.MarketConsumer.EventListener(ctx)
	go r.LiquidateConsumer.Run(ctx)
	go r.ApiResyncRoutine(ctx)
	<-ctx.Done()
}

func Wrapper(conf config.Config, filters api.MarketFilters) {
	conn := connector.New(conf.Endpoints)
	result, err := api.QueryMarkets(conf.ChainID)
	if err != nil {
		fmt.Println(err)
		return
	}
	markets := api.FilterMarket(result, filters, conf.ChainID)
	cached := cache.NewCache(conf, markets, filters)

	// pass empty
	routeCache := swap.NewRouteCache()
	runn := NewRunner(conn, conf, routeCache, cached)
	runn.Init(context.Background())
	runn.Run(context.Background())

}
