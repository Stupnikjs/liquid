package cache

import (
	"context"
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
			positions := ApiItemToPos(fetched, id)
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
			})

		}(id)
	}

	wg.Wait()
	return firstErr
}

func ApiItemToPos(result api.PositionsResult, marketId [32]byte) []*BorrowPosition {
	var positions []*BorrowPosition
	for _, p := range result.MarketPositions.Items {

		pos := &BorrowPosition{
			BorrowShares:     utils.ParseBigInt(p.State.BorrowShares.String()),
			CollateralAssets: utils.ParseBigInt(p.State.Collateral.String()),
			MarketID:         marketId,
			Address:          common.HexToAddress(p.User.Address),
		}
		positions = append(positions, pos)
	}
	return positions
}

func ParsePositions(id [32]byte, result api.PositionsResult) []BorrowPosition {
	items := result.MarketPositions.Items // ✅ plus de .Data
	positions := make([]BorrowPosition, 0, len(items))
	for _, item := range items {
		borrowAssetUsd := utils.ParseBigInt(item.State.BorrowAssetsUsd.String())
		if borrowAssetUsd.Cmp(utils.TenPowInt(2)) < 0 {
			continue
		}
		borrowShares := utils.ParseBigInt(item.State.BorrowShares.String())
		collateral := utils.ParseBigInt(item.State.Collateral.String())
		if borrowShares.Sign() == 0 && collateral.Sign() == 0 {
			continue
		}
		positions = append(positions, BorrowPosition{
			MarketID:         id,
			Address:          common.HexToAddress(item.User.Address),
			BorrowShares:     borrowShares,
			CollateralAssets: collateral,
		})
	}
	return positions
}
