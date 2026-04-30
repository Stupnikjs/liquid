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

const rpcTimeout = 5 * time.Second

// OnChainResult holds the raw values returned from on-chain calls.
type OnChainResult struct {
	ID          [32]byte
	Stats       market.MarketStats
	OraclePrice *big.Int
}

func newResult(id [32]byte) *OnChainResult {
	return &OnChainResult{
		ID:          id,
		Stats:       market.MarketStats{},
		OraclePrice: new(big.Int),
	}
}

// marketCall builds the eth_call for a Morpho market.
func marketCall(morphoAddr common.Address, id [32]byte, res *OnChainResult) w3types.RPCCaller {
	return eth.CallFunc(morphoAddr, config.MarketFunc, id).Returns(
		new(big.Int), new(big.Int),
		&res.Stats.TotalBorrowAssets,
		&res.Stats.TotalBorrowShares,
		new(big.Int),
		new(big.Int),
	)
}

// oracleCall builds the eth_call for an oracle price.
func oracleCall(oracle common.Address, res *OnChainResult) w3types.RPCCaller {
	return eth.CallFunc(oracle, config.OraclePriceFunc).Returns(res.OraclePrice)
}

// refresh is the shared implementation for on-chain refreshes.
// callBuilder returns the list of RPCCallers to batch, plus a function
// that writes the results back into the market cache.
func refresh(
	conn *connector.Connector,
	ctx context.Context,
	id [32]byte,
	callBuilder func() ([]w3types.RPCCaller, func()),
) error {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	calls, apply := callBuilder()
	if err := conn.EthCallCtx(ctx, calls); err != nil {
		fmt.Printf("[onchain] rpc error %x: %v\n", id[:4], err)
		return err
	}
	apply()
	return nil
}

// OnChainRefresh fetches both the market state and the oracle price.
func OnChainRefresh(
	conn *connector.Connector, ctx context.Context,
	c state.MarketReader, mParam morpho.MarketParams,
	id [32]byte, morphoAddr common.Address,
) error {
	return refresh(conn, ctx, id, func() ([]w3types.RPCCaller, func()) {
		res := newResult(id)
		calls := []w3types.RPCCaller{
			marketCall(morphoAddr, id, res),
			oracleCall(mParam.Oracle, res),
		}
		apply := func() {
			c.Update(res.ID, func(m *market.Market) {
				m.Stats.TotalBorrowAssets = res.Stats.TotalBorrowAssets
				m.Stats.TotalBorrowShares = res.Stats.TotalBorrowShares
				m.Oracle.Price = res.OraclePrice
			})
		}
		return calls, apply
	})
}

// OnChainOracleRefresh fetches only the oracle price.
func OnChainOracleRefresh(
	conn *connector.Connector, ctx context.Context,
	c state.MarketReader, mParam morpho.MarketParams,
	id [32]byte, morphoAddr common.Address,
) error {
	return refresh(conn, ctx, id, func() ([]w3types.RPCCaller, func()) {
		res := newResult(id)
		calls := []w3types.RPCCaller{oracleCall(mParam.Oracle, res)}
		apply := func() {
			c.Update(res.ID, func(m *market.Market) {
				m.Oracle.Price = res.OraclePrice
			})
		}
		return calls, apply
	})
}
