package core

import (
	"context"
	"fmt"
	"math/big"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

type OnChainResult struct {
	ID [32]byte

	Stats  market.MarketStats
	Oracle *big.Int
}

func OnChainCalls(c *Cache) ([]w3types.RPCCaller, map[int][32]byte, []*OnChainResult) {
	var calls []w3types.RPCCaller

	ids := c.Markets.Ids()
	results := make([]*OnChainResult, 0, len(ids))

	callIndexToID := make(map[int][32]byte)

	for _, id := range ids {

		res := &OnChainResult{
			ID:     id,
			Stats:  market.MarketStats{},
			Oracle: new(big.Int),
		}

		results = append(results, res)

		// market call
		callIdx := len(calls)
		callIndexToID[callIdx] = id

		calls = append(calls,
			eth.CallFunc(config.MorphoMain, config.MarketFunc, id).Returns(
				new(big.Int), new(big.Int),
				&res.Stats.TotalBorrowAssets,
				&res.Stats.TotalBorrowShares,
				new(big.Int),
				new(big.Int),
			),
		)
		snap := c.Markets.GetSnapshot(id)
		if snap != nil {
			calls = append(calls,
				eth.CallFunc(c.Markets.GetSnapshot(id).Oracle.Address, config.OraclePriceFunc).
					Returns(res.Oracle),
			)
		}

		// oracle call

	}
	return calls, callIndexToID, results
}

func OnChainRefresh(conn *connector.Connector, c *Cache) error {
	ctx := context.Background()
	calls, _, results := OnChainCalls(c)
	if err := conn.EthCallCtx(ctx, calls); err != nil {
		return err
	}
	ApplyResults(c, results)
	return nil
}

func ApplyResults(c *Cache, results []*OnChainResult) {
	for _, r := range results {
		c.Markets.Update(r.ID, func(m *market.Market) {
			m.Stats = r.Stats
			m.Oracle.Price = r.Oracle
		})
	}
}

/* refactor */
func CheckSlippage(c *Cache, client *w3.Client, ctx context.Context, m morpho.MarketParams, seizeAssets *big.Int, oraclePrice *big.Int) (float64, error) {
	params := api.QuoteParams{
		TokenIn:           m.CollateralToken,
		TokenOut:          m.LoanToken,
		AmountIn:          seizeAssets,
		Fee:               big.NewInt(int64(m.PoolFee)),
		SqrtPriceLimitX96: big.NewInt(0),
	}

	var amountOut *big.Int
	var sqrtPriceAfter *big.Int
	var ticksCrossed uint32
	var gasEst *big.Int

	if err := client.CallCtx(ctx,
		eth.CallFunc(config.BaseUniswapV3Router, config.FuncQuoteExactInputSingle, params).Returns(
			&amountOut, &sqrtPriceAfter, &ticksCrossed, &gasEst,
		),
	); err != nil {
		return 0, fmt.Errorf("quote failed: %w", err)
	}

	// Prix oracle : ce qu'on s'attend à recevoir
	expectedOut := new(big.Int).Mul(seizeAssets, oraclePrice)
	expectedOut.Div(expectedOut, utils.TenPowInt(36))

	// Slippage = (expectedOut - amountOut) / expectedOut
	diff := new(big.Int).Sub(expectedOut, amountOut)
	slippage := utils.BigIntToFloat(diff) / utils.BigIntToFloat(expectedOut)

	return slippage, nil
}
