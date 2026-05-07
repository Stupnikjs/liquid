package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/Stupnikjs/liquid/internal/liquidate"
	"github.com/Stupnikjs/liquid/internal/utils"
)

func (r *Runner) OnChainRefreshRoutine(ctx context.Context) {
	for _, id := range r.Cache.Markets.Ids() {
		go r.MarketRoutine(ctx, id)
	}
}

func (r *Runner) ApiCall(ctx context.Context) error {
	return r.Cache.ApiCall(r.Conn, uint32(r.Config.ChainID))
}

func (r *Runner) ApiResyncRoutine(ctx context.Context) {
	utils.RunTicker(ctx, 30*time.Minute, func() {
		if err := r.ApiCall(ctx); err != nil {
			r.log(fmt.Sprintf("api resync error: %v", err))
		}
	})
}

func (r *Runner) LiquidationRoutine(ctx context.Context) {
	consumer := &liquidate.Consumer{
		Conn:      r.Conn,
		Cache:     r.Cache.Markets,
		MarketMap: r.Cache.MarketMap,
		Logger:    r.Logger,
		Signer:    r.Config.Signer,
		Ch:        r.LiquidateCh,
	}
	consumer.Run(ctx)

}

func (r *Runner) log(msg string) {
	select {
	case r.Logger <- msg:
	default:
	}
}
