package runner

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/api"
)

func (r *MarketConsumer) ApiResyncRoutine(ctx context.Context) {
	utils.RunTicker(ctx, 30*time.Minute, func() {
		if err := r.ApiCall(); err != nil {
			log.Printf("api resync error: %v", err)
		}
	})
}

func (r *MarketConsumer) ApiCall() error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	ctx := context.Background()
	for _, id := range r.Cache.Ids() {
		wg.Add(1)
		go func(id [32]byte) {
			defer wg.Done()
			fetched, err := api.FetchAllPositions(ctx, id, r.Config.ChainID)
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

				if p.BorrowAssetsUsd.Cmp(utils.WAD_5) < 0 || p.BorrowAssetsUsd.Cmp(utils.WAD_10_000) > 0 {
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
			r.Cache.UpdatePositionsSlice(id, positions)

			r.Cache.UpdateMaxCollateralPos(id, positions[0].CollateralAssets)

		}(id)
	}

	wg.Wait()
	return firstErr
}
