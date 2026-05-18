package liquidate

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
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
func (c *Consumer) ComputeAmounts(m morpho.MarketParams, snap *cache.MarketSnapshot, p *cache.BorrowPosition) *Liquidable {
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
	route, ok := c.SwapCache.GetRoute(m.CollateralToken, m.LoanToken)
	if !ok {
		return out
	}
	maxSwapAmout := route.WCAmountOut
	if seizeAssets.Cmp(maxSwapAmout) > 0 {
		seizeAssets = new(big.Int).Set(maxSwapAmout) // chercher a terme le swap manager ici
	}

	out.SeizeAssets = seizeAssets
	minOut := morpho.ComputeMinOut(seizeAssets, snap.Oracle.Price, snap.Oracle.Price)
	out.MinOut = minOut
	return out
}

// refactor on multihop
func (c *Consumer) ToLiquidationArg(l *Liquidable, fee int64, params morpho.MarketParams) (lqtypes.LiquidateArgs, error) {
	route, ok := c.SwapCache.GetRoute(params.CollateralToken, params.LoanToken)

	if !ok {
		return lqtypes.LiquidateArgs{}, fmt.Errorf("no avaible route found")
	}
	if l.MinOut.Cmp(route.Hops[0].WCAmountOut) > 0 {
		l.MinOut.Set(route.Hops[0].WCAmountOut)
	}

	/*
		swapSteps := EncodeRouteToCallData()




	*/
	return lqtypes.LiquidateArgs{
		MarketParams: *params.ToMarketContractParams(),
		Borrower:     l.Pos.Address,
		SeizedAssets: l.SeizeAssets,
		RepaidShares: big.NewInt(0),
		SwapRouter:   route.Hops[0].Router,
		PoolFee:      big.NewInt(int64(route.Hops[0].Fee)),
		MinOut:       l.MinOut,
	}, nil
}

// snap nil guard upon this func
// err is passed to Liquidable
func (c *Consumer) SimulateAndPreComputeTx(ctx context.Context, m morpho.MarketParams, snap *cache.MarketSnapshot, fee int64, p *cache.BorrowPosition) (*Liquidable, error) {
	l := c.ComputeAmounts(m, snap, p)
	args, err := c.ToLiquidationArg(l, fee, m)
	if err != nil {
		return l, fmt.Errorf("to liquidation args err : %w", err)

	}
	data, err := EncodeLiquidateCalldata(args)
	if err != nil {
		return l, fmt.Errorf("encode: %w", err)

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
