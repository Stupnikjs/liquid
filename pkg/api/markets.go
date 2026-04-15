package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

type QuoteParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	AmountIn          *big.Int
	Fee               *big.Int
	SqrtPriceLimitX96 *big.Int
}

type MarketConfig struct {
	morpho.MarketParams
}

func (m MarketItem) ToConfig(chainid uint32) MarketConfig {
	return MarketConfig{
		morpho.MarketParams{
			ID:                      [32]byte(common.HexToHash(m.UniqueKey)),
			ChainID:                 chainid,
			LoanToken:               common.HexToAddress(m.LoanAsset.Address),
			LoanTokenStr:            m.LoanAsset.Symbol,
			CollateralToken:         common.HexToAddress(m.CollateralAsset.Address),
			CollateralTokenStr:      m.CollateralAsset.Symbol,
			Oracle:                  common.HexToAddress(m.OracleAddress),
			LLTV:                    utils.ParseBigInt(string(m.Lltv)),
			LoanTokenDecimals:       uint16(m.LoanAsset.Decimals),
			CollateralTokenDecimals: uint16(m.CollateralAsset.Decimals),
			PoolFee:                 0,
			Correlated:              false,
			CexOnly:                 false,
		},
	}
}
func FilterMarket(client *w3.Client, chainid uint32) []MarketConfig {
	ctx := context.Background()

	var result MarketsResult
	if err := Query(ctx, MarketsQuery(chainid), &result); err != nil {
		fmt.Printf("graphql fetch: %s", err.Error())
		return nil
	}

	const MinBorrowUsd = 15_000.0
	var mark []MarketConfig
	for _, m := range result.Markets.Items {
		supplyUsd, _ := m.State.SupplyAssetsUsd.Float64()
		borrowUsd, _ := m.State.BorrowAssetsUsd.Float64()

		if supplyUsd == 0 {
			continue
		}

		if borrowUsd < 10_000 || borrowUsd > 100_000 || borrowUsd/supplyUsd < 0.1 {
			continue
		}

		mark = append(mark, m.ToConfig(chainid))
	}

	return mark

}

// Call all pos from market with id
func FetchBorrowersFromMarket(marketId [32]byte, chainId uint32) ([]market.BorrowPosition, error) {
	ctx := context.Background()
	marketID := "0x" + hex.EncodeToString(marketId[:])

	fmt.Println("fetching market:", marketID) // ← ajoute ça

	var result PositionsResult
	err := Query(ctx, PositionsQuery(marketID, uint32(chainId)), &result)
	if err != nil {
		return nil, fmt.Errorf("graphql fetch: %w", err)
	}

	fmt.Println("items returned:", len(result.MarketPositions.Items)) // ← et ça

	positions := ParsePositions(marketId, result)
	fmt.Println("positions parsed:", len(positions))
	return positions, nil
}

// Wrapper for api call
func ApiCall(client *w3.Client, marketR state.MarketReader, marketMap map[[32]byte]morpho.MarketParams, chainId uint32) error {
	var wg sync.WaitGroup
	errs := make(chan error, len(marketMap))

	for id := range marketMap {
		wg.Add(1)
		go func(id [32]byte) {
			defer wg.Done()
			fetched, err := FetchBorrowersFromMarket(id, chainId)
			if err != nil {
				errs <- err
				return
			}
			for _, p := range fetched {
				marketR.Update(id, func(m *market.Market) {
					m.Positions[p.Address] = &p
				})
			}
		}(id)
	}
	wg.Wait()
	close(errs)
	return <-errs // retourne la première erreur s'il y en a une
}

func ParsePositions(id [32]byte, result PositionsResult) []market.BorrowPosition {
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
	return positions
}
