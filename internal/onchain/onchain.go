package onchain

import (
	"context"
	"math/big"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

/* sortir dans un package séparé avec event.go */
type OnChainResult struct {
	ID          [32]byte
	Stats       market.MarketStats
	OraclePrice *big.Int
}

func OnChainCalls(c state.MarketReader, markets map[[32]byte]morpho.MarketParams, OnlyOracle bool) ([]w3types.RPCCaller, map[int][32]byte, []*OnChainResult) {
	var calls []w3types.RPCCaller

	ids := c.Ids()
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
		if !OnlyOracle {
			calls = append(calls,
				eth.CallFunc(config.MorphoMain, config.MarketFunc, id).Returns(
					new(big.Int), new(big.Int),
					&res.Stats.TotalBorrowAssets,
					&res.Stats.TotalBorrowShares,
					new(big.Int),
					new(big.Int),
				),
			)
		}

		calls = append(calls,
			eth.CallFunc(markets[id].Oracle, config.OraclePriceFunc).
				Returns(res.OraclePrice),
		)
	}

	// oracle call

	return calls, callIndexToID, results
}

func OnChainRefresh(conn *connector.Connector, c state.MarketReader, markets map[[32]byte]morpho.MarketParams, onlyOracle bool) error {
	ctx := context.Background()
	calls, _, results := OnChainCalls(c, markets, onlyOracle)
	if err := conn.EthCallCtx(ctx, calls); err != nil {
		return err
	}
	ApplyResults(c, results, onlyOracle)
	return nil
}

func ApplyResults(c state.MarketReader, results []*OnChainResult, onlyOracle bool) {
	for _, r := range results {
		c.Update(r.ID, func(m *market.Market) {
			if !onlyOracle {
				m.Stats = r.Stats
			}
			m.Oracle.Price = r.OraclePrice
		})
	}
}
