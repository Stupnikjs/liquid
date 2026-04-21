package runner

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/liquidate"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
)

func (r *Runner) OnChainRefreshRoutine(ctx context.Context) {
	for _, id := range r.Cache.Markets.Ids() {
		go r.MarketRoutine(ctx, id)
	}

}

func distanceToInterval(distance float64) time.Duration {
	switch {
	// 1% if ETH pair < 0.0001
	case distance < 0.01:
		return 2 * time.Second
	// 1% if ETH pair < 0.0003
	case distance < 0.03:
		return 10 * time.Second
	// 1% if ETH pair < 0.0005
	case distance < 0.05:
		return 100 * time.Second
	default:
		return 200 * time.Second
	}
}

func (r *Runner) MarketRoutine(ctx context.Context, id [32]byte) {
	// Wait for initial data
	var snap *market.MarketSnapshot
	for snap == nil || snap.Oracle.Price == nil || snap.Oracle.Price.Sign() == 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
			snap = r.Cache.Markets.GetSnapshot(id)
		}
	}
	morphoM := r.Cache.MarketMap[id]
	info := state.CheckMarket(r.Cache.Markets, morphoM)
	interval := distanceToInterval(info.PerctToFirstLiq)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			onchain.OnChainRefresh(r.Conn, r.Cache.Markets, r.Cache.GetMorphoMarketFromId(id), id)
			// HF CHECKS HERE
			fmt.Println(r.Cache.Markets.GetSnapshot(id))
			info = state.CheckMarket(r.Cache.Markets, morphoM)
			if len(info.Liquidables) > 0 {
				for _, l := range info.Liquidables {
					r.LiquidateCh <- l.Pos
				}

			}
			newInterval := distanceToInterval(info.PerctToFirstLiq)
			if info.IsETHCorrelated() {
				newInterval = distanceToInterval(info.PerctToFirstLiq * 100)
			}
			// BUILDING REPORT
			if info.Snap.Oracle.Price == nil {
				break
			}
			r.Logger <- state.MarketReport(info)
			if newInterval != interval {
				ticker.Reset(newInterval)
				interval = newInterval
			}
		}
	}
}

func (r *Runner) ApiCallRoutine(ctx context.Context) error {
	return r.Cache.ApiCall(r.Conn.ClientHTTP, uint32(r.Config.ChainID))
}

func (r *Runner) SubscribePositionRoutine(ctx context.Context) {
	r.Conn.SubscribeToEventPos(ctx)
}

func (r *Runner) LogEthCallsPerMin(ctx context.Context) {
	r.Conn.LogsEthCallsFromLastMin(ctx, r.Logger)
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

				liqPos := &liquidate.Liquidable{Pos: &p}
				result := liquidate.SimulateAndPreComputeTx(r.Conn, r.Cache.Markets, r.Cache.MarketMap, liqPos)

				if result.SimErr != nil {
					r.Logger <- fmt.Sprintf("[liq] simulation failed for %s: %v", p.Address, result.SimErr)
					return
				}

				if !result.IsLiquidable || result.EstProfit.Sign() <= 0 {
					r.Logger <- fmt.Sprintf("[liq] not profitable for %s profit=%s", p.Address, result.EstProfit)
					return
				}

				r.Logger <- fmt.Sprintf("[liq] sending tx for %s profit=%s gas=%d", p.Address, result.EstProfit, result.GasEstimate)
				morphoP := r.Cache.MarketMap[p.MarketID]
				err := liquidate.LiquidateCall(r.Config.Signer, r.Conn.ClientHTTP, ctx, liquidate.LiquidateArgs{
					MarketParams: *morphoP.ToMarketContractParams(),
					Borrower:     p.Address,
					SeizedAssets: result.SeizeAssets,
					RepaidShares: result.RepayShares,
					SwapRouter:   liquidate.SwapRouterAddr,
					PoolFee:      big.NewInt(int64(morphoP.PoolFee)), // fees not set here
				})
				if err != nil {
					r.Logger <- fmt.Sprintf("[liq] tx failed for %s: %v", p.Address, err)
					return
				}

				r.Logger <- fmt.Sprintf("[liq] ✓ liquidated %s", p.Address)
			}()
		}
	}
}
