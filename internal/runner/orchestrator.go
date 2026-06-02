package runner

import (
	"context"
	"fmt"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/pkg/api"
	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/Stupnikjs/liquid/pkg/swap"
)

// logs ethcalls missing

func (r *Runner) Run(ctx context.Context) {
	go r.SubscribePositionRoutine(ctx)
	go r.OnChainRefreshRoutine(ctx)
	go r.EventListener(ctx)
	go r.LiquidationRoutine(ctx)
	go r.ApiResyncRoutine(ctx)
	<-ctx.Done()
}

func Wrapper(conf lqtypes.Config, filters api.MarketFilters, logfile string) {
	conn := connector.New(conf.Endpoints)
	result, err := api.QueryMarkets(conf.ChainID)
	if err != nil {

		fmt.Println(err)
		return
	}
	markets := api.FilterMarket(result, filters, conf.ChainID)

	cached := cache.NewCache(conf, markets, filters)
	infra := &lqtypes.Infra{
		Conn:   conn,
		Config: conf,
	}
	// pass empty
	routeCache := swap.NewRouteCache()
	runn := NewRunner(infra, routeCache, cached, logfile)
	runn.Init(context.Background())
	runn.Run(context.Background())

}
