package runner

import (
	"context"

	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
)

func (r *Runner) SubscribePositionRoutine(ctx context.Context) {
	r.Conn.SubscribeToEventPos(ctx)
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
