package runner

import (
	"context"
)

func (r *Runner) Run(ctx context.Context) {

	go r.SubscribePositionRoutine(ctx)
	// rpc calls per market => market routines
	go r.OnChainRefreshRoutine(ctx)
	// rpc call per minutes
	go r.LogEthCallsNum(ctx)
	// log markets info
	go r.EventListener(ctx)
	go r.LiquidationRoutine(ctx)

	go r.ApiResyncRoutine(ctx)
	// 👇 bloque proprement
	<-ctx.Done()
}
