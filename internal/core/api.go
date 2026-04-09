package core

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/position"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/lmittmann/w3"
)

// Call all pos from market with id
func FetchBorrowersFromMarket(marketId [32]byte, chainId uint32) ([]position.BorrowPosition, error) {
	ctx := context.Background()
	marketID := "0x" + hex.EncodeToString(marketId[:])
	var result api.PositionsResult
	err := api.Query(ctx, api.PositionsQuery(marketID, uint32(chainId)), &result)
	if err != nil {
		return nil, fmt.Errorf("graphql fetch: %w", err)
	}

	return position.ParsePositions(marketId, result), nil
}

// Wrapper for api call that filters out pos with big HF
func (c *Cache) ApiRefreshCache(client *w3.Client, threshold *big.Int) error {

	for _, id := range c.Markets.Ids() {
		fetched, err := FetchBorrowersFromMarket(id, 8453)
		if err != nil {
			return err
		}

		snap := c.Markets.GetSnapshot(id)
		for _, p := range fetched {

			stats := snap.Stats
			// filter here
			hf := p.HF(stats.TotalBorrowShares, stats.TotalBorrowAssets, snap.Oracle.Price, snap.LLTV)
			if hf.Cmp(threshold) < 0 {
				c.Markets.Update(id, func(m *market.Market) {
					m.Positions[p.Address] = &p
				})

			}

		}

	}

	return nil
}
