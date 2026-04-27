package cache

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

func (c *Cache) ApiCall(client *w3.Client, chainId uint32) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	ctx := context.Background()
	// gets positions and maxPos for swap
	for id := range c.MarketMap {
		wg.Add(1)
		go func(id [32]byte) {
			defer wg.Done()
			fetched, err := api.FetchAllPositions(ctx, id, chainId)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			var positions []*BorrowPosition
			for _, pos := range fetched {
				p := ApiItemToPos(pos, id)
				// pos less than 1 dollard
				if p.BorrowAssetsUsd.Cmp(utils.WAD) < 0 {
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
			c.Markets.Update(id, func(m *Market) {
				m.Positions = positions
			})

			c.Markets.Update(id, func(m *Market) {
				m.Stats.MaxCollateralPos = new(big.Int).Set(positions[0].CollateralAssets)
				fmt.Println(new(big.Int).Set(positions[0].CollateralAssets))
			})

		}(id)
	}

	wg.Wait()
	return firstErr
}

func ApiItemToPos(p api.PositionItem, marketId [32]byte) *BorrowPosition {
	return &BorrowPosition{
		BorrowShares:     utils.ParseBigInt(p.State.BorrowShares.String()),
		BorrowAssetsUsd:  utils.ParseBigInt(p.State.BorrowAssetsUsd.String()),
		CollateralAssets: utils.ParseBigFloatToBigInt(p.State.Collateral.String()),
		MarketID:         marketId,
		Address:          common.HexToAddress(p.User.Address),
	}

}
