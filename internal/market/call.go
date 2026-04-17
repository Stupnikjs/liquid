package market

import (
	"math/big"
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

	// gets positions and maxPos for swap
	for id := range c.MarketMap {
		wg.Add(1)
		go func(id [32]byte) {
			defer wg.Done()

			fetched, err := api.FetchBorrowersFromMarket(id, chainId)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			maxPos := big.NewInt(0)
			positions := ApiItemToPos(fetched, id)
			for _, p := range positions {
				cp := *p
				cp.CollateralAssets = new(big.Int).Set(p.CollateralAssets) // deep copy

				if cp.CollateralAssets.Cmp(maxPos) > 0 {
					maxPos.Set(cp.CollateralAssets)
				}
				addr := p.Address // capture explicite
				c.Markets.Update(id, func(m *Market) {
					m.Positions[addr] = &cp
				})

			}
			c.Markets.Update(id, func(m *Market) {
				m.Stats.MaxCollateralPos = new(big.Int).Set(maxPos)
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
