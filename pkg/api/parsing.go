package api

import (
	"encoding/json"
	"fmt"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/ethereum/go-ethereum/common"
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

func ParsePositions(id [32]byte, result PositionsResult) []market.BorrowPosition {
	fmt.Println("len items: ", len(result.MarketPositions.Items))
	items := result.MarketPositions.Items // ✅ plus de .Data
	positions := make([]market.BorrowPosition, 0, len(items))

	for _, item := range items {
		/*
			borrowAssetUsd := utils.ParseBigInt(item.State.BorrowAssetsUsd.String())

				if borrowAssetUsd.Cmp(utils.TenPowInt(3)) < 0 {
					continue
				}*/
		borrowShares := utils.ParseBigInt(item.State.BorrowShares.String())
		collateral := utils.ParseBigInt(item.State.Collateral.String())

		if borrowShares.Sign() == 0 && collateral.Sign() == 0 {
			continue
		}

		positions = append(positions, market.BorrowPosition{
			MarketID:         id,
			Address:          common.HexToAddress(item.User.Address),
			BorrowShares:     borrowShares,
			CollateralAssets: collateral,
		})
	}
	fmt.Println("position out of parse pos: ", len(positions))
	return positions
}
