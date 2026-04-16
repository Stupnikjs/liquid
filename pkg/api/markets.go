package api

import (
	"context"
	"fmt"
	"math/big"

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

func QueryMarkets(client *w3.Client, chainid uint32) (MarketsResult, error) {

	ctx := context.Background()

	var result MarketsResult
	if err := Query(ctx, MarketsQuery(chainid), &result); err != nil {
		fmt.Printf("graphql fetch: %s", err.Error())
		return MarketsResult{}, nil
	}
	return result, nil
}

type MarketFilters struct {
	MinUsdMarket float64
	MaxUsdMarket float64
}

func FilterMarket(result MarketsResult, filters MarketFilters, chainid uint32) []morpho.MarketParams {

	var mark []morpho.MarketParams
	for _, m := range result.Markets.Items {
		supplyUsd, _ := m.State.SupplyAssetsUsd.Float64()
		borrowUsd, _ := m.State.BorrowAssetsUsd.Float64()

		if supplyUsd == 0 {
			continue
		}

		if borrowUsd < filters.MinUsdMarket || borrowUsd > filters.MaxUsdMarket {
			continue
		}

		mark = append(mark, m.ToConfig(chainid).MarketParams)
	}

	return mark

}
