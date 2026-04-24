package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// Call all pos from market with id
// Wrapper for api call

// Dans types.go ou en haut de api.go
type PositionItem struct {
	User struct {
		Address string `json:"address"`
	} `json:"user"`
	State struct {
		BorrowShares    json.Number `json:"borrowShares"`
		BorrowAssetsUsd json.Number `json:"borrowAssetsUsd"`
		Collateral      json.Number `json:"collateral"`
	} `json:"state"`
}

// PositionsResult mis à jour pour utiliser PositionItem
type PositionsResult struct {
	MarketPositions struct {
		Items    []PositionItem `json:"items"`
		PageInfo struct {
			CountTotal  int    `json:"countTotal"`
			EndCursor   string `json:"endCursor"`
			HasNextPage bool   `json:"hasNextPage"`
		} `json:"pageInfo"`
	} `json:"marketPositions"`
}

// FetchAllPositions devient propre
func FetchAllPositions(ctx context.Context, marketID string, chainID uint32) ([]PositionItem, error) {
	var all []PositionItem
	cursor := ""

	for {
		var result PositionsResult
		if err := Query(ctx, PositionsQuery(marketID, chainID, cursor), &result); err != nil {
			return nil, fmt.Errorf("fetch positions page cursor=%q: %w", cursor, err)
		}

		mp := result.MarketPositions
		all = append(all, mp.Items...)

		if !mp.PageInfo.HasNextPage || mp.PageInfo.EndCursor == "" {
			break
		}
		cursor = mp.PageInfo.EndCursor
	}

	return all, nil
}
