package runner

import (
	"context"

	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
)

func (r *Runner) Run(ctx context.Context) {

	go r.SubscribePositionRoutine(ctx)
	// rpc calls per market => market routines
	go r.OnChainRefreshRoutine(ctx)
	// rpc call per minutes
	go r.LogEthCallsPerMin(ctx)
	// log markets info
	go r.EventListener(ctx)
	// 👇 bloque proprement
	<-ctx.Done()
}

func (r *Runner) EventListener(ctx context.Context) {
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
