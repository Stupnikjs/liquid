package onchain

import (
	"context"
	"fmt"
	"math/big"

	market "github.com/Stupnikjs/morpho-sepolia/internal/cache"
	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

type OnChainResult struct {
	ID          [32]byte
	Stats       market.MarketStats
	OraclePrice *big.Int
}

func OnChainCalls(c state.MarketReader, mParam morpho.MarketParams, id [32]byte) ([]w3types.RPCCaller, map[int][32]byte, *OnChainResult) {
	var calls []w3types.RPCCaller

	callIndexToID := make(map[int][32]byte)

	res := &OnChainResult{
		ID:          id,
		Stats:       market.MarketStats{},
		OraclePrice: new(big.Int),
	}

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
		eth.CallFunc(mParam.Oracle, config.OraclePriceFunc).
			Returns(res.OraclePrice),
	)

	// oracle call

	return calls, callIndexToID, res
}

func OnChainRefresh(conn *connector.Connector, c state.MarketReader, mParam morpho.MarketParams, id [32]byte) error {
	ctx := context.Background()
	calls, _, results := OnChainCalls(c, mParam, id)
	if err := conn.EthCallCtx(ctx, calls); err != nil {
		fmt.Printf("[onchain] rpc error %x: %v\n", id[:4], err)
		return err
	}
	ApplyResults(c, results)
	return nil
}

func ApplyResults(c state.MarketReader, results *OnChainResult) {

	c.Update(results.ID, func(m *market.Market) {
		m.Stats.TotalBorrowAssets = results.Stats.TotalBorrowAssets
		m.Stats.TotalBorrowShares = results.Stats.TotalBorrowShares
		m.Oracle.Price = results.OraclePrice

	})

}
