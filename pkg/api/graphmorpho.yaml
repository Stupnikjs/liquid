package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	MorphoGraphQLURL = "https://api.morpho.org/graphql"
	DefaultPageLimit = 200
)

// ── HTTP ─────────────────────────────────────────────────────────────────────

func GraphqlPost(query string) ([]byte, error) {
	body, err := json.Marshal(GraphQLRequest{Query: query})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(MorphoGraphQLURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// ── QUERIES ──────────────────────────────────────────────────────────────────

func PositionsQuery(marketID string, chainID uint32) string {
	return fmt.Sprintf(`{
        marketPositions(
			first:1000
            where: {
                marketUniqueKey_in: ["%s"]
                chainId_in: [%d]
            }
        ) {
            items {
                user { address }
                state { borrowShares collateral }
            }
        }
    }`, marketID, chainID)
}

// ── FETCH ────────────────────────────────────────────────────────────────────

// ── TYPES ────────────────────────────────────────────────────────────────────

type GraphQLRequest struct {
	Query string `json:"query"`
}

type GraphQLResult struct {
	Data struct {
		MarketPositions struct {
			Items []struct {
				User struct {
					Address string `json:"address"`
				} `json:"user"`
				State struct {
					BorrowShares json.Number `json:"borrowShares"`
					Collateral   json.Number `json:"collateral"`
				} `json:"state"`
			} `json:"items"`
			PageInfo struct {
				CountTotal int `json:"countTotal"`
			} `json:"pageInfo"`
		} `json:"marketPositions"`
	} `json:"data"`
}
