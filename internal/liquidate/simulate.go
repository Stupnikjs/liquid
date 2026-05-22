package liquidate

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/internal/swap"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

// dryRun performs a batched eth_call + gas estimation against the liquidator
// contract without submitting a transaction. Returns the gas estimate on
// success, or an error if the call reverts (position not liquidable).
func (c *Consumer) dryRun(ctx context.Context, data []byte) (gasVal uint64, err error) {

	msg := w3types.Message{
		From:  c.Infra.Config.Addresses.Wallet,
		To:    &c.Infra.Config.Addresses.LiquidatorContract,
		Input: data,
	}

	var callResult []byte

	if err := c.Infra.Conn.FallBackEthCallCtx(ctx, []w3types.RPCCaller{
		eth.Call(&msg, nil, nil).Returns(&callResult),
		eth.EstimateGas(&msg, nil).Returns(&gasVal),
	}); err != nil {
		return 0, fmt.Errorf("dryRun: %w", err)
	}

	return gasVal, nil
}

// Gets snapshot
// Compute liquidation values
// Encode args
// Simulate with eth_call
func (c *Consumer) ComputeAmounts(m morpho.MarketParams, snap *cache.MarketSnapshot, p *cache.BorrowPosition, maxSwapAmount *big.Int) *Liquidable {
	out := &Liquidable{}
	out.Pos = p

	// 1. Math pure — pas de RPC
	repayShares, seizeAssets := morpho.ComputeLiquidationAmounts(
		p.BorrowShares,
		snap.Stats.TotalBorrowAssets,
		snap.Stats.TotalBorrowShares,
		snap.LLTV,
	)
	out.RepayShares = repayShares

	// changer pour le multihop

	if seizeAssets.Cmp(maxSwapAmount) > 0 {
		seizeAssets = new(big.Int).Set(maxSwapAmount) // chercher a terme le swap manager ici
	}

	out.SeizeAssets = seizeAssets
	minOut := morpho.ComputeMinOut(seizeAssets, snap.Oracle.Price, snap.Oracle.Price)
	out.MinOut = minOut
	return out
}

// refactor on multihop
func (c *Consumer) ToLiquidationArg(l *Liquidable, params morpho.MarketParams, route []lqtypes.PoolEdge) ([]byte, error) {
	m := params.ToMarketContractParams()
	steps, err := BuildSteps(route, c.Infra.Config.Addresses.LiquidatorContract)
	if err != nil {
		return nil, fmt.Errorf("build steps: %w", err)
	}
	return BuildLiquidateCalldata(*m, l.Pos.Address, l.SeizeAssets, l.RepayShares, steps, new(big.Int).SetInt64(0))
}

// snap nil guard upon this func
// err is passed to Liquidable
func (c *Consumer) SimulateAndPreComputeTx(ctx context.Context, m morpho.MarketParams, snap *cache.MarketSnapshot, p *cache.BorrowPosition) (*Liquidable, error) {
	routes := c.SwapCache.FindRoutes(m.CollateralToken, m.LoanToken, 3)
	route, err := swap.BestRoute(routes, big.NewInt(0))
	maxSwapAmount := new(big.Int).Set(route[0].WCAmountIn)
	l := c.ComputeAmounts(m, snap, p, maxSwapAmount)
	if len(routes) == 0 {
		return l, fmt.Errorf("route is len 0")
	}

	data, err := c.ToLiquidationArg(l, m, route)
	if err != nil {
		return l, fmt.Errorf("to liquidation arg encoding err : %w", err)

	}

	l.CallData = data
	gasVal, err := c.dryRun(ctx, data)
	l.SimulatedAt = time.Now()
	if err != nil {
		return l, fmt.Errorf("eth_call failed: %w", err)
	}
	l.GasEstimate = gasVal
	c.log(fmt.Sprintf("seized asset %s with successfull simulation  %s", utils.FormatDecimals(l.SeizeAssets, int(m.CollateralTokenDecimals)), utils.FormatWAD(l.Pos.CachedHF)))
	l.IsLiquidable = true
	return l, nil
}
