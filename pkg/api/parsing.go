package api

import (
	"encoding/json"
	"fmt"
)

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
                state { 
borrowShares borrowAssetsUsd collateral }
            }
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
				BorrowShares    json.Number `json:"borrowShares"`
				BorrowAssetsUsd json.Number `json:"borrowAssetsUsd"`
				Collateral      json.Number `json:"collateral"`
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
