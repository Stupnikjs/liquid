package api

import (
	"encoding/json"
	"fmt"
)

func PositionsQuery(marketID string, chainID uint32, skip int) string {
	return fmt.Sprintf(`{
        marketPositions(
            first: 1000
            skip: %d
            where: {
                marketUniqueKey_in: ["%s"]
                chainId_in: [%d]
            }
        ) {
            items {
                user { address }
                state {
                    borrowShares borrowAssetsUsd collateral
                }
            }
            pageInfo {
                count
                countTotal
            }
        }
    }`, skip, marketID, chainID)
}

func MarketsQuery(chainid uint32) string {
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
type PageInfo struct {
	Count      int `json:"count"`
	CountTotal int `json:"countTotal"`
}

// PositionsResult mis à jour pour utiliser PositionItem
type PositionsResult struct {
	MarketPositions struct {
		Items    []PositionItem `json:"items"`
		PageInfo PageInfo       `json:"pageInfo"`
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
