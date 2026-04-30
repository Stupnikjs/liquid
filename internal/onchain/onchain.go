package onchain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	market "github.com/Stupnikjs/morpho-sepolia/internal/cache"
	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

//

type OnChainResult struct {
	ID          [32]byte
	Stats       market.MarketStats
	OraclePrice *big.Int
}

/*
	split to reduce only oracle calls
	and reduce market call
*/

func OnChainCalls(c state.MarketReader, mParam morpho.MarketParams, id [32]byte, morphoAddr common.Address) ([]w3types.RPCCaller, *OnChainResult) {
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
		eth.CallFunc(morphoAddr, config.MarketFunc, id).Returns(
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

	return calls, res
}

func OnChainRefresh(conn *connector.Connector, ctx context.Context, c state.MarketReader, mParam morpho.MarketParams, id [32]byte, morphoAddr common.Address) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	calls, results := OnChainCalls(c, mParam, id, morphoAddr)

	if err := conn.EthCallCtx(ctx, calls); err != nil {
		fmt.Printf("[onchain] rpc error %x: %v\n", id[:4], err)
		return err
	}

	ApplyResults(c, results)
	return nil
}

func OnChainOracleRefresh(conn *connector.Connector, ctx context.Context, c state.MarketReader, mParam morpho.MarketParams, id [32]byte, morphoAddr common.Address) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	call, results := OnChainOracleCall(c, mParam, id, morphoAddr)

	if err := conn.EthSingleCallCtx(ctx, call); err != nil {
		fmt.Printf("[onchain] rpc error %x: %v\n", id[:4], err)
		return err
	}

	ApplyOracle(c, results)
	return nil
}

func OnChainOracleCall(c state.MarketReader, mParam morpho.MarketParams, id [32]byte, morphoAddr common.Address) (w3types.RPCCaller, *OnChainResult) {
	res := &OnChainResult{
		ID:          id,
		Stats:       market.MarketStats{},
		OraclePrice: new(big.Int),
	}

	call := eth.CallFunc(mParam.Oracle, config.OraclePriceFunc).
		Returns(res.OraclePrice)

	// oracle call

	return call, res
}

func ApplyResults(c state.MarketReader, results *OnChainResult) {
	// fmt.Println("results: ", results.Stats.TotalBorrowAssets, results.Stats.TotalBorrowShares, results.OraclePrice)
	c.Update(results.ID, func(m *market.Market) {

		m.Stats.TotalBorrowAssets = results.Stats.TotalBorrowAssets
		m.Stats.TotalBorrowShares = results.Stats.TotalBorrowShares
		m.Oracle.Price = results.OraclePrice

	})

}

func ApplyOracle(c state.MarketReader, results *OnChainResult) {
	// fmt.Println("results: ", results.Stats.TotalBorrowAssets, results.Stats.TotalBorrowShares, results.OraclePrice)
	c.Update(results.ID, func(m *market.Market) {
		m.Oracle.Price = results.OraclePrice

	})

}
