package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/cache"
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
	sem := make(chan struct{}, 5)

	for {
		select {
		case <-ctx.Done():
			return
		case pos := <-r.LiquidateCh:
			p := pos
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			go func() {
				defer func() { <-sem }()
				r.LiquidateWrapper(ctx, &p)
			}()
		}
	}
}

func (r *Runner) LiquidateWrapper(ctx context.Context, p *cache.BorrowPosition) {

	// Precompute and simulation
	result := liquidate.SimulateAndPreComputeTx(r.Conn, r.Cache.Markets, r.Cache.MarketMap, p)
	if result.SimErr != nil {
		r.Logger <- fmt.Sprintf("[liq] simulation failed for %s: %v", p.Address, result.SimErr)
		return
	}
	if !result.IsLiquidable {
		r.Logger <- fmt.Sprintf("[liq] not profitable for %s profit=%s", p.Address, result.EstProfit)
		return
	}
	r.Logger <- fmt.Sprintf("[liq] sending tx for %s profit=%s gas=%d", p.Address, result.EstProfit, result.GasEstimate)

	// Simlation worked now send the tx
	err := liquidate.LiquidateCall(r.Config.Signer, r.Conn.ClientHTTP, ctx, result.Args)

	if err != nil {
		r.Logger <- fmt.Sprintf("[liq] tx failed for %s: %v", p.Address, err)
		return
	}
	r.Logger <- fmt.Sprintf("[liq] ✓ liquidated %s", p.Address)
}
