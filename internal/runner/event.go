package runner

import (
	"context"

	"github.com/Stupnikjs/liquid/internal/onchain"
)

func (r *Runner) SubscribePositionRoutine(ctx context.Context) {
	r.Infra.Conn.SubscribeToEventPos(ctx, r.Infra.Config)
}

func (r *Runner) EventListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-r.Infra.Conn.LogsCh():
			if !ok {
				return
			}
			onchain.ProcessEvents(r.Store.MarketReader, event)
		}
	}
}
