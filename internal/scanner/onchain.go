package scanner

import (
	"context"
	"math/big"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
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
				eth.CallFunc(snap.Oracle.Address, config.OraclePriceFunc).
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

/*

func GetSlippageBps(
	ctx context.Context,
	conn *connector.Connector,
	m morpho.MarketParams,
	seizeAssets *big.Int,
	oraclePrice *big.Int,
) (int64, error) {

	params := api.QuoteParams{
		TokenIn:           m.CollateralToken,
		TokenOut:          m.LoanToken,
		AmountIn:          seizeAssets,
		Fee:               big.NewInt(int64(m.PoolFee)),
		SqrtPriceLimitX96: big.NewInt(0),
	}

	amountOut, err := QuoteUniswap(ctx, conn, params)
	if err != nil {
		return 0, err
	}

	expected := ComputeExpectedOut(seizeAssets, oraclePrice)

	return ComputeSlippageBps(expected, amountOut)
}

func QuoteUniswap(
	ctx context.Context,
	conn *connector.Connector,
	params api.QuoteParams,
) (amountOut *big.Int, err error) {

	var sqrtPriceAfter *big.Int
	var ticksCrossed uint32
	var gasEst *big.Int

	err = conn.EthSingleCallCtx(ctx,
		eth.CallFunc(config.BaseUniswapV3Router, config.FuncQuoteExactInputSingle, params).Returns(
			&amountOut, &sqrtPriceAfter, &ticksCrossed, &gasEst,
		),
	)

	if err != nil {
		return nil, fmt.Errorf("uniswap quote failed: %w", err)
	}

	return amountOut, nil
}

func ComputeExpectedOut(
	seizeAssets *big.Int,
	oraclePrice *big.Int,
) *big.Int {
	out := new(big.Int).Mul(seizeAssets, oraclePrice)
	return out.Div(out, utils.TenPowInt(36))
}

func ComputeSlippageBps(expected, actual *big.Int) (int64, error) {
	if expected.Sign() == 0 {
		return 0, fmt.Errorf("expectedOut is zero")
	}

	diff := new(big.Int).Sub(expected, actual)

	// (diff / expected) * 10_000  (bps)
	bps := new(big.Int).Mul(diff, big.NewInt(10_000))
	bps.Div(bps, expected)

	return bps.Int64(), nil
}
*/
