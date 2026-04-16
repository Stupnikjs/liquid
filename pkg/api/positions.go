package api

import (
	"context"
	"encoding/hex"
	"fmt"
)

// Call all pos from market with id
// Wrapper for api call

func FetchBorrowersFromMarket(marketId [32]byte, chainId uint32) (PositionsResult, error) {

	ctx := context.Background()
	marketID := "0x" + hex.EncodeToString(marketId[:])
	var result PositionsResult
	err := Query(ctx, PositionsQuery(marketID, uint32(chainId)), &result)
	if err != nil {
		return PositionsResult{}, fmt.Errorf("graphql fetch: %w", err)
	}

	return result, nil
}
