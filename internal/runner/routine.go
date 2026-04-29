package runner

import (
	"context"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/liquidate"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
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

func (r *Runner) LogMarketState(ctx context.Context) {
	utils.RunTicker(ctx, time.Minute, func() {
		r.Cache.Markets.Range(func(id [32]byte) {
			morphoM := r.Cache.GetMorphoMarketFromId(id)
			r.Logger <- state.GetMarketLog(r.Cache.Markets, id, morphoM)
		})
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
