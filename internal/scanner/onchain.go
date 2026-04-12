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


/* sortir dans un package séparé avec event.go */
type OnChainResult struct {
	ID          [32]byte
	Stats       market.MarketStats
	OraclePrice *big.Int
}

func OnChainCalls(c *Cache) ([]w3types.RPCCaller, map[int][32]byte, []*OnChainResult) {
	var calls []w3types.RPCCaller

	ids := c.Markets.Ids()
	results := make([]*OnChainResult, 0, len(ids))

	callIndexToID := make(map[int][32]byte)

	for _, id := range ids {

		res := &OnChainResult{
			ID:          id,
			Stats:       market.MarketStats{},
			OraclePrice: new(big.Int),
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

		calls = append(calls,
			eth.CallFunc(c.marketMap[id].Oracle, config.OraclePriceFunc).
				Returns(res.OraclePrice),
		)
	}

	// oracle call

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
			m.Oracle.Price = r.OraclePrice
		})
	}
}
