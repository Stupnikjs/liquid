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
func (c *Consumer) ComputeAmounts(m morpho.MarketParams, p *cache.BorrowPosition) *Liquidable {
	out := &Liquidable{}
	out.Pos = p
	snap := c.Store.MarketReader.GetSnapshot(p.MarketID)
	if snap == nil {
		out.SimErr = fmt.Errorf("snap nil")
		return out
	}

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
		seizeAssets = maxSwapAmout // chercher a terme le swap manager ici
	}

	out.SeizeAssets = seizeAssets
	minOut := morpho.ComputeMinOut(seizeAssets, snap.Oracle.Price, snap.Oracle.Price)
	out.MinOut = minOut
	return out
}

func (c *Consumer) ToLiquidationArg(l *Liquidable, fee int64, params morpho.MarketParams) lqtypes.LiquidateArgs {
	return lqtypes.LiquidateArgs{
		MarketParams: *params.ToMarketContractParams(),
		Borrower:     l.Pos.Address,
		SeizedAssets: l.SeizeAssets,
		RepaidShares: big.NewInt(0),
		SwapRouter:   c.Infra.Config.Addresses.UniSwapRouter,
		PoolFee:      big.NewInt(int64(fee)),
		MinOut:       l.MinOut,
	}
}

func (c *Consumer) SimulateAndPreComputeTx(ctx context.Context, m morpho.MarketParams, fee int64, p *cache.BorrowPosition) *Liquidable {
	l := c.ComputeAmounts(m, p)
	args := c.ToLiquidationArg(l, fee, m)
	data, err := args.EncodeLiquidateCalldata()
	if err != nil {
		l.SimErr = fmt.Errorf("encode: %w", err)
		return l
	}
	l.CallData = data

	gasVal, err := c.dryRun(ctx, data)
	l.SimulatedAt = time.Now()
	if err != nil {
		l.SimErr = fmt.Errorf("eth_call failed: %w", err)
		l.IsLiquidable = false
		return l
	}
	l.GasEstimate = gasVal
	c.log(fmt.Sprintf("seized asset %s with successfull simulation  %s", utils.FormatDecimals(l.SeizeAssets, int(m.CollateralTokenDecimals)), utils.FormatWAD(l.Pos.CachedHF)))
	l.IsLiquidable = true
	return l
}
