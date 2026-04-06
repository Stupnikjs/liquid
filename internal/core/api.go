package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/lmittmann/w3"
)

/* Api call */

func FetchBorrowersFromMarket(param morpho.MarketParams) ([]BorrowPosition, error) {
	marketID := "0x" + hex.EncodeToString(param.ID[:])

	data, err := api.GraphqlPost(api.PositionsQuery(marketID, param.ChainID))
	if err != nil {
		return nil, fmt.Errorf("graphql fetch: %w", err)
	}

	var result api.GraphQLResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("graphql decode: %w", err)
	}

	return parsePositions(param, result), nil
}

func (c *Cache) ApiRefreshCache(client *w3.Client) error {

	for _, ma := range c.Config.Markets {
		market := c.PositionCache.m[ma.ID]
		fetched, err := FetchBorrowersFromMarket(ma)
		if err != nil {
			return err
		}

		market.Mu.Lock()
		for _, p := range fetched {

			market.Positions[p.Address] = &p

		}
		market.Mu.Unlock()
	}

	return nil
}
