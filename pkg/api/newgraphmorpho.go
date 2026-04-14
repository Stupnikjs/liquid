package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	MorphoGraphQLURL = "https://api.morpho.org/graphql"
)

// ── CLIENT ───────────────────────────────────────────────────────────────────

// Query reste inchangé, c'est une implémentation générique solide.
func Query(ctx context.Context, query string, out any) error {
	body, err := json.Marshal(struct {
		Query string `json:"query"`
	}{Query: query})
	if err != nil {
		return fmt.Errorf("marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, MorphoGraphQLURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []GraphQLError  `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return envelope.Errors[0]
	}

	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}
	return nil
}

type GraphQLError struct {
	Message string `json:"message"`
}

func (e GraphQLError) Error() string { return "graphql: " + e.Message }

// ── QUERIES ──────────────────────────────────────────────────────────────────

func PositionsQuery(marketID string, chainID uint32) string {
	return fmt.Sprintf(`{
        marketPositions(
            first: 1000
            where: {
                marketUniqueKey_in: ["%s"]
                chainId_in: [%d]
            }
        ) {
            items {
                user { address }
                state { borrowShares collateral }
            }
            pageInfo { countTotal }
        }
    }`, marketID, chainID)
}

func MarketsQuery(chainid uint32) string {
	// On récupère tout sur Base (8453) pour filtrer ensuite en Go
	return fmt.Sprintf(`{
        markets(
            orderBy: SupplyAssetsUsd
            orderDirection: Desc
            first: 1000
            where: { chainId_in: [%d] }
        ) {
            items {
                uniqueKey
                creationTimestamp
                oracleAddress
                lltv
                loanAsset {
                    address
                    symbol
                    decimals
                }
                collateralAsset {
                    address
                    symbol
                    decimals
                }
                state {
                    supplyAssetsUsd
                    borrowAssetsUsd
                }
            }
        }
    }`, chainid)
}

// ── TYPES ────────────────────────────────────────────────────────────────────

type PositionsResult struct {
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
}

type MarketsResult struct {
	Markets struct {
		Items []MarketItem `json:"items"`
	} `json:"markets"`
}

type MarketItem struct {
	UniqueKey         string      `json:"uniqueKey"`
	CreationTimestamp int64       `json:"creationTimestamp"`
	OracleAddress     string      `json:"oracleAddress"`
	Lltv              json.Number `json:"lltv"`
	LoanAsset         Asset       `json:"loanAsset"`
	CollateralAsset   *Asset      `json:"collateralAsset"` // Pointeur car peut être null
	State             struct {
		SupplyAssetsUsd json.Number `json:"supplyAssetsUsd"`
		BorrowAssetsUsd json.Number `json:"borrowAssetsUsd"`
	} `json:"state"`
}

type Asset struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}
