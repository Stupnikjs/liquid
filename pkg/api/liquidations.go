package api

import (
	"context"
	"fmt"
)

func FetchAllLiquidations(ctx context.Context, chainID uint32) ([]LiquidationItem, error) {
	client := NewHTTPClient()
	var all []LiquidationItem
	skip := 0

	for {
		var result LiquidationsResult
		if err := client.Query(ctx, LiquidationsQuery(chainID, skip), &result); err != nil {
			return nil, fmt.Errorf("fetch liquidations page skip=%d: %w", skip, err)
		}

		tx := result.Transactions
		all = append(all, tx.Items...)

		// for debug
		if skip > 5000 {
			break
		}
		skip += tx.PageInfo.Count
		if skip >= tx.PageInfo.CountTotal {
			break
		}
	}

	return all, nil
}
