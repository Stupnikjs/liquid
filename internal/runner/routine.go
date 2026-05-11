package runner

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/liquidate"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/api"
)

func (r *Runner) OnChainRefreshRoutine(ctx context.Context) {
	for _, id := range r.Store.MarketReader.Ids() {
		go r.MarketRoutine(ctx, id)
	}
}

func (r *Runner) ApiResyncRoutine(ctx context.Context) {
	utils.RunTicker(ctx, 30*time.Minute, func() {
		if err := r.ApiCall(); err != nil {
			r.log(fmt.Sprintf("api resync error: %v", err))
		}
	})
}

func (r *Runner) LiquidationRoutine(ctx context.Context) {
	consumer := &liquidate.Consumer{
		Infra:  r.Infra,
		Store:  r.Store,
		Logger: r.Logger,
		Ch:     r.LiquidateCh,
	}
	consumer.Run(ctx)

}

func (r *Runner) log(msg string) {
	select {
	case r.Logger <- msg:
	default:
	}
}

func (r *Runner) ApiCall() error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	ctx := context.Background()
	for _, id := range r.Store.MarketReader.Ids() {
		wg.Add(1)
		go func(id [32]byte) {
			defer wg.Done()
			fetched, err := api.FetchAllPositions(ctx, id, r.Infra.Config.ChainID)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			var positions []*cache.BorrowPosition
			for _, pos := range fetched {
				p := cache.ApiItemToPos(pos, id)
				// pos less than 1 dollard
				if p.BorrowAssetsUsd.Cmp(utils.WAD_10) < 0 || p.BorrowAssetsUsd.Cmp(utils.WAD_10_000) > 0 {
					continue
				}
				positions = append(positions, p)
			}
			sort.Slice(positions, func(i, j int) bool {
				pi := positions[i].CollateralAssets
				pj := positions[j].CollateralAssets
				// nil traité comme zéro → rejeté en fin
				if pi == nil && pj == nil {
					return false
				}
				if pi == nil {
					return false
				}
				if pj == nil {
					return true
				}
				return pi.Cmp(pj) > 0
			})
			if len(positions) == 0 {
				return
			}
			// maybe sorting by collateral here
			r.Store.MarketReader.Update(id, func(m *cache.Market) {
				m.Positions = positions
			})

			r.Store.MarketReader.Update(id, func(m *cache.Market) {
				m.Stats.MaxCollateralPos = new(big.Int).Set(positions[0].CollateralAssets)
			})

		}(id)
	}

	wg.Wait()
	return firstErr
}
