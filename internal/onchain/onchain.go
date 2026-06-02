package onchain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	market "github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/config/abi"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/Stupnikjs/liquid/pkg/morpho"
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
	return eth.CallFunc(morphoAddr, abi.MarketFunc, id).Returns(
		new(big.Int), new(big.Int),
		&res.Stats.TotalBorrowAssets,
		&res.Stats.TotalBorrowShares,
		new(big.Int),
		new(big.Int),
	)
}

// oracleCall builds the eth_call for an oracle price.
func oracleCall(oracle common.Address, res *OnChainResult) w3types.RPCCaller {
	return eth.CallFunc(oracle, abi.OraclePriceFunc).Returns(res.OraclePrice)
}

// refresh is the shared implementation for on-chain refreshes.
// callBuilder returns the list of RPCCallers to batch, plus a function
// that writes the results back into the market cache.
func refresh(
	conn connector.Connector,
	ctx context.Context,
	id [32]byte,
	callBuilder func() ([]w3types.RPCCaller, func()),
) error {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	calls, apply := callBuilder()
	// oracle call must be fast (alchemy)
	if err := conn.SecondCallCtx(ctx, calls...); err != nil {
		fmt.Printf("[onchain] rpc error %x: %v\n", id[:4], err)
		return err
	}
	apply()
	return nil
}

// OnChainRefresh fetches both the market state and the oracle price.
func OnChainRefresh(
	infra *lqtypes.Infra, ctx context.Context,
	c *cache.Cache, mParam morpho.MarketParams,
	id [32]byte,
) error {
	return refresh(infra.Conn, ctx, id, func() ([]w3types.RPCCaller, func()) {
		res := newResult(id)
		calls := []w3types.RPCCaller{
			marketCall(infra.Config.Addresses.Morpho, id, res),
			oracleCall(mParam.Oracle, res),
		}
		apply := func() {
			c.Markets.Update(res.ID, func(m *market.Market) {
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
	infra *lqtypes.Infra, ctx context.Context,
	c *cache.Cache, mParam morpho.MarketParams,
	id [32]byte, morphoAddr common.Address,
) error {
	return refresh(infra.Conn, ctx, id, func() ([]w3types.RPCCaller, func()) {
		res := newResult(id)
		calls := []w3types.RPCCaller{
			oracleCall(mParam.Oracle, res),
		}
		apply := func() {
			c.Markets.Update(res.ID, func(m *market.Market) {
				// log price to debug
				m.Oracle.Price = res.OraclePrice
			})
		}
		return calls, apply
	})
}
