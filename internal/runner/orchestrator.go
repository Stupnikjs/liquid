package runner

import (
	"context"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/connector"
	"github.com/Stupnikjs/liquid/pkg/api"
	"github.com/Stupnikjs/liquid/pkg/config"
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

func Wrapper(conf config.Config, filters api.MarketFilters, logfile string) {
	conn := connector.New(conf.RPC.HTTP[0], conf.RPC.HTTP[1], conf.RPC.WS[0])
	cached := cache.NewCache(conf, filters)
	runn := NewRunner(cached, conn, conf, logfile)
	runn.Init(context.Background())
	runn.Run(context.Background())

}
