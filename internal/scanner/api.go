package core

import (
	"context"
	"encoding/hex"
	"fmt"

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

// Wrapper for api call
func (c *Cache) ApiCall(client *w3.Client) error {

	for id := range c.marketMap {

		fetched, err := FetchBorrowersFromMarket(id, 8453)
		if err != nil {
			return err
		}
		for _, p := range fetched {
			c.Markets.Update(id, func(m *market.Market) {
				m.Positions[p.Address] = &p
			})
		}
	}

	return nil
}
