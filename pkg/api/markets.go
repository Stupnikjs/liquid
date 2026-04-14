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

		if borrowUsd < 10_000 || borrowUsd > 1_000_000 || borrowUsd/supplyUsd < 0.1 {
			continue
		}
		mark = append(mark, m.ToConfig(chainid))
	}

	return mark

}
