package runner

import (
	"context"
)

func (r *Runner) Run(ctx context.Context) {
	go r.SubscribePositionRoutine(ctx)
	go r.OnChainRefreshRoutine(ctx)
	go r.EventListener(ctx)
	go r.LiquidationRoutine(ctx)
	go r.ApiResyncRoutine(ctx)
	<-ctx.Done()
}
