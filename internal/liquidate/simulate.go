package liquidate

import (
	"context"
	"fmt"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
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
func (c *Consumer) ComuteAmounts(p *market.BorrowPosition) *Liquidable {
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
	if seizeAssets.Cmp(snap.Stats.MaxUniSwappable) > 0 {
		seizeAssets = snap.Stats.MaxUniSwappable // chercher a terme le swap manager ici
	}

	out.SeizeAssets = seizeAssets
	minOut := morpho.ComputeMinOut(seizeAssets, snap.Oracle.Price, snap.Oracle.Price)
	out.MinOut = minOut
}

func (c *Consumer) ToLiquidationArg(l *Liquidable, fee uint32, params morpho.MarketParams) lqtypes.LiquidateArgs {
	return lqtypes.LiquidateArgs{
		MarketParams: *params.ToMarketContractParams(),
		Borrower:     l.Pos.Address,
		SeizedAssets: l.SeizeAssets,
		RepaidShares: big.NewInt(0),
		SwapRouter:   c.Infra.Config.Addresses.UniSwapRouter,
		PoolFee:      fee,
		MinOut:       l.MinOut,
	}
}

func (c *Consumer) SimulateAndPreComputeTx(ctx context.Context, p *cache.BorrowPosition) *Liquidable {
	l := c.ComputeAmounts(p)
	args := c.ToLiquidationArg(l, snap, params)
	params := c.Store.MarketMap[p.MarketID]
	data, err := args.EncodeLiquidateCalldata()
	if err != nil {
		out.SimErr = fmt.Errorf("encode: %w", err)
		return out
	}
	out.CallData = data

	gasVal, err := c.dryRun(ctx, data)
	out.SimulatedAt = time.Now()
	if err != nil {
		out.SimErr = fmt.Errorf("eth_call failed: %w", err)
		out.IsLiquidable = false
		return out
	}
	out.GasEstimate = gasVal
	c.log(fmt.Sprintf("seized asset %s with successfull simulation  %s", utils.FormatDecimals(out.SeizeAssets, int(params.CollateralTokenDecimals)), utils.FormatWAD(out.Pos.CachedHF)))
	out.IsLiquidable = true
	return out
}
