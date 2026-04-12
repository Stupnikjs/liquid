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
func FilterMarket(client *w3.Client) []MarketConfig {
	ctx := context.Background()

	var result MarketsResult
	if err := Query(ctx, MarketsQuery(), &result); err != nil {
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

		if borrowUsd < 10_000 || borrowUsd > 100_000_000 || borrowUsd/supplyUsd < 0.1 {
			continue
		}
		mark = append(mark, m.ToConfig())
	}
	/*
		usdc := common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
		weth := common.HexToAddress("0x4200000000000000000000000000000000000006")

		amountIn := big.NewInt(100 * 1e6) // 100 USDC


		// aller : USDC → WETH
		out1, err := QuoteSwap(client, usdc, weth, amountIn, 500)
		fmt.Printf("USDC→WETH: %s err: %v\n", out1.AmountOut, err)

		// retour : WETH → USDC avec ce qu'on a reçu
		out2, err := QuoteSwap(client, weth, usdc, out1.AmountOut, 500)
		fmt.Printf("WETH→USDC: %s err: %v\n", out2.AmountOut, err)

		// slippage round trip
		diff := new(big.Int).Sub(amountIn, out2.AmountOut)
		fmt.Printf("round trip loss: %s USDC (sur %s)\n", diff, amountIn)
	*/
	return mark

}
