package liquidate

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/Stupnikjs/liquid/pkg/swap"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

// snap nil guard upon this func
// err is passed to Liquidable
func (c *Consumer) SimulateAndPreComputeTx(ctx context.Context, m morpho.MarketParams, snap *cache.MarketSnapshot, p *cache.BorrowPosition) (*Liquidable, error) {
	routes := c.SwapCache.FindRoutes(m.CollateralToken, m.LoanToken, 3)
	route, err := swap.BestRoute(routes, big.NewInt(0))
	if err != nil {
		return nil, fmt.Errorf("err finding best route: %w", err)
	}

	maxSwapAmount, err := swap.RouteMaxAmountIn(route)
	if err != nil {
		return nil, fmt.Errorf("error calculating max swap amount: %w", err)
	}
	log.Printf("maxswap amount: %s \n", maxSwapAmount.String())
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
	log.Printf("seized asset %s with successfull simulation  %s", utils.FormatDecimals(l.SeizeAssets, int(m.CollateralTokenDecimals)), utils.FormatWAD(l.Pos.CachedHF))
	l.IsLiquidable = true
	return l, nil
}

// dryRun performs a batched eth_call + gas estimation against the liquidator
// contract without submitting a transaction. Returns the gas estimate on
// success, or an error if the call reverts (position not liquidable).
func (c *Consumer) dryRun(ctx context.Context, data []byte) (gasVal uint64, err error) {

	msg := w3types.Message{
		From:  c.Config.Addresses.Wallet,
		To:    &c.Config.Addresses.LiquidatorContract,
		Input: data,
	}

	var callResult []byte

	calls := []w3types.RPCCaller{
		eth.Call(&msg, nil, nil).Returns(&callResult),
		eth.EstimateGas(&msg, nil).Returns(&gasVal)}

	if err := c.Conn.SecondCallCtx(ctx, calls...); err != nil {
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
	seizeAssets := morpho.ComputeSeizedAsset(
		p.BorrowShares,
		snap.Stats.TotalBorrowAssets,
		snap.Stats.TotalBorrowShares,
		snap.LLTV,
	)
	out.RepayShares = 0
	if seizeAssets.Cmp(maxSwapAmount) > 0 {
		seizeAssets = new(big.Int).Set(maxSwapAmount)
	}
	out.SeizeAssets = seizeAssets
	minOut := morpho.CollateralAssetsInLoan(seizeAssets, snap.Oracle.Price)
	out.MinOut = minOut
	return out
}
