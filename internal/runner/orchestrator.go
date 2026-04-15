package runner

import (
	"context"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
)

func (r *Runner) Run(ctx context.Context) {

	go r.WatchPositionRoutine(ctx)
	// Onchain rpc pool to update markets
	go r.OnChainRefreshRoutine(ctx)
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

func distanceToInterval(distance float64) time.Duration {
	switch {
	// 1%
	case distance < 0.01:
		return 1 * time.Second

	case distance < 0.03:
		return 10 * time.Second
	case distance < 0.05:
		return 100 * time.Second
	default:
		return 200 * time.Second
	}
}
