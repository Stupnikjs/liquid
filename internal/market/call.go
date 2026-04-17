package market

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/swap"
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
			for _, p := range ApiItemToPos(fetched, id) {
				if p.CollateralAssets.Cmp(maxPos) > 0 {
					maxPos = p.CollateralAssets
				}
				c.Markets.Update(id, func(m *Market) {
					m.Positions[p.Address] = p
				})

			}
			if maxPos.Sign() != 0 {
				c.Markets.Update(id, func(m *Market) {
					m.Stats.MaxCollateralPos = maxPos
				})
			}

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

func (c *Cache) CheckSlipage(conn *connector.Connector) {

	c.Markets.Range(func(id [32]byte) {
		morphoMarket := c.MarketMap[id]
		snap := c.Markets.GetSnapshot(id)
		fmt.Println(snap)
		if snap == nil {
			return
		}
		out, err := swap.BestQuote(conn, morphoMarket.CollateralToken, morphoMarket.LoanToken, snap.Stats.MaxCollateralPos)

		if out == nil {
			return
		}
		maxPosFloat, _ := snap.Stats.MaxCollateralPos.Float64()
		outFloat, _ := out.AmountOut.Float64()
		slippage := (1 - outFloat/maxPosFloat) * 100
		fmt.Println(morphoMarket.CollateralTokenStr, morphoMarket.LoanTokenStr, slippage, err)
	})
}
