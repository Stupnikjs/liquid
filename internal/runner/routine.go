package runner

import (
	"context"

	"github.com/Stupnikjs/morpho-sepolia/internal/liquidate"
)

/*          Parralel calls in Orchestrator                         */

func (r *Runner) OnChainRefreshRoutine(ctx context.Context) {
	for _, id := range r.Cache.Markets.Ids() {
		go r.MarketRoutine(ctx, id)
	}

}

func (r *Runner) ApiCallRoutine(ctx context.Context) error {
	return r.Cache.ApiCall(r.Conn.ClientHTTP, uint32(r.Config.ChainID))
}

func (r *Runner) LogEthCallsPerMin(ctx context.Context) {
	r.Conn.LogsEthCallsFromLastMin(ctx, r.Logger)
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
