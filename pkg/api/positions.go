package api

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/lmittmann/w3"
)

// Call all pos from market with id
func FetchBorrowersFromMarket(marketId [32]byte, chainId uint32) ([]market.BorrowPosition, error) {

	ctx := context.Background()
	marketID := "0x" + hex.EncodeToString(marketId[:])
	fmt.Println(marketID)
	var result PositionsResult
	err := Query(ctx, PositionsQuery(marketID, uint32(chainId)), &result)
	if err != nil {
		return nil, fmt.Errorf("graphql fetch: %w", err)
	}
	positions := ParsePositions(marketId, result)
	return positions, nil
}

// Wrapper for api call
func ApiCall(client *w3.Client, marketR state.MarketReader, marketMap map[[32]byte]morpho.MarketParams, chainId uint32) error {

	for id := range marketMap {
		fetched, _ := FetchBorrowersFromMarket(id, chainId)
		for _, p := range fetched {
			marketR.Update(id, func(m *market.Market) {
				m.Positions[p.Address] = &p
			})
		}

	}
	return nil

}
