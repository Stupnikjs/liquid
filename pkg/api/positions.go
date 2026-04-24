package api

import (
	"context"
	"fmt"
)

// Call all pos from market with id
// Wrapper for api call

// FetchAllPositions devient propre
func FetchAllPositions(ctx context.Context, marketID [32]byte, chainID uint32) ([]PositionItem, error) {
	var all []PositionItem
	skip := 0
	strId := fmt.Sprintf("0x%x", marketID)
	for {
		var result PositionsResult
		if err := Query(ctx, PositionsQuery(strId, chainID, skip), &result); err != nil {
			return nil, fmt.Errorf("fetch positions page skip=%d: %w", skip, err)
		}

		mp := result.MarketPositions
		all = append(all, mp.Items...)

		skip += mp.PageInfo.Count
		if skip >= mp.PageInfo.CountTotal {
			break
		}
	}

	return all, nil
}
