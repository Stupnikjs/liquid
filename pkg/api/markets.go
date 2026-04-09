package api

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"time"

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

func (m MarketItem) ToConfig() MarketConfig {
	return MarketConfig{
		morpho.MarketParams{
			ID:                      [32]byte(common.HexToHash(m.UniqueKey)),
			ChainID:                 8453,
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

func LogHotMarket(client *w3.Client, lastXmounth int) []MarketConfig {

	ctx := context.Background()

	var result MarketsResult
	err := Query(ctx, MarketsQuery(), &result)
	if err != nil {
		fmt.Printf("graphql fetch: %s", err.Error())
	}

	markets := result.Markets.Items
	sort.Slice(markets, func(i, j int) bool {
		return markets[i].CreationTimestamp > markets[j].CreationTimestamp
	})
	end := []MarketItem{}
	for _, m := range markets {
		if m.State.BorrowAssetsUsd < 50000 {
			continue
		}
		if m.CreationTimestamp > time.Now().AddDate(0, -1*lastXmounth, 0).Unix() {
			end = append(end, m)
		}

	}
	sort.Slice(end, func(i, j int) bool {
		return end[i].State.BorrowAssetsUsd > end[j].State.BorrowAssetsUsd
	})

	mark := []MarketConfig{}
	for _, m := range end {
		mark = append(mark, m.ToConfig())
	}
	return mark
}
