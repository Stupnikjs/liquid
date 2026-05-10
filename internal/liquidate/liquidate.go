package liquidate

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/connector"
	"github.com/Stupnikjs/liquid/internal/onchain"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/config"
	"github.com/Stupnikjs/liquid/pkg/lqtypes"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

// mockable dans testutil
type EthCaller interface {
	EthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error
}

type Liquidable struct {
	Pos          *cache.BorrowPosition
	RepayShares  *big.Int
	SeizeAssets  *big.Int
	MinOut       *big.Int
	EstProfit    *big.Int
	GasEstimate  uint64
	SimulatedAt  time.Time
	SimErr       error
	IsLiquidable bool
	Args         lqtypes.LiquidateArgs
}

func NewConsumer(conn *connector.Connector, marketReader cache.MarketReader, marketMap map[[32]byte]morpho.MarketParams, config config.Config, logger chan string, ch <-chan cache.BorrowPosition) *Consumer {
	return &Consumer{
		Conn:      conn,
		Cache:     marketReader,
		MarketMap: marketMap,
		Config:    config,
		Logger:    logger,
		Ch:        ch,
	}
}

type Consumer struct {
	Conn      lqtypes.EthCaller
	Cache     cache.MarketReader
	MarketMap map[[32]byte]morpho.MarketParams
	Config    config.Config
	Logger    chan string
	Ch        <-chan cache.BorrowPosition
}

func (c *Consumer) log(msg string) {
	select {
	case c.Logger <- msg:
	default: // drop if full — never block the liquidation path
	}
}

// Pure math, zero RPC — unit testable
func (c *Consumer) ToLiquidationArg(l *Liquidable, snap *cache.MarketSnapshot, params morpho.MarketParams, minOut *big.Int) lqtypes.LiquidateArgs {
	return lqtypes.LiquidateArgs{
		MarketParams: *params.ToMarketContractParams(),
		Borrower:     l.Pos.Address,
		SeizedAssets: l.SeizeAssets,
		RepaidShares: big.NewInt(0),
		SwapRouter:   c.Config.Addresses.UniSwapRouter,
		PoolFee:      big.NewInt(int64(snap.Stats.SwapFee)),
		MinOut:       minOut,
	}
}

// ABI encode — testable isolément

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

	if err := c.Conn.FallBackEthCallCtx(ctx, []w3types.RPCCaller{
		eth.Call(&msg, nil, nil).Returns(&callResult),
		eth.EstimateGas(&msg, nil).Returns(&gasVal),
	}); err != nil {
		return 0, fmt.Errorf("dryRun: %w", err)
	}

	return gasVal, nil
}

func (c *Consumer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case pos := <-c.Ch:
			c.liquidateWrapper(ctx, c.Cache, &pos)
		}
	}
}

func (c *Consumer) liquidateWrapper(ctx context.Context, mReader cache.MarketReader, p *cache.BorrowPosition) {
	result := c.SimulateAndPreComputeTx(ctx, mReader, p)
	if result.SimErr != nil {
		c.log(fmt.Sprintf("[liq] simulation failed for %s: %v", p.Address, result.SimErr))
		return
	}
	if !result.IsLiquidable {
		c.log("[liq] not profitable")
		return
	}
	market := c.MarketMap[p.MarketID]
	c.log(fmt.Sprintf("[liq] sending tx for %s seized=%s market %s ", p.Address, utils.FormatDecimals(result.SeizeAssets, int(market.CollateralTokenDecimals)), market.GetPair()))

	err := c.LiquidateCall(ctx, result.Args, result.GasEstimate)
	if err != nil {
		c.log(fmt.Sprintf("[liq] tx failed for %s: %v", p.Address, err))
		return
	}
	c.log(fmt.Sprintf("[liq] ✓ liquidated pair%s marketid:%s borrower:%s", market.GetPair(), hexutil.Encode(market.ID[:]), p.Address))
}

func ComputeMinOut(seizedAssets, collateralPrice, loanPrice *big.Int) *big.Int {
	valueInLoan := new(big.Int).Mul(seizedAssets, collateralPrice)
	valueInLoan.Div(valueInLoan, loanPrice)
	return valueInLoan
}

func (c *Consumer) SimulateAndPreComputeTx(ctx context.Context, mReader cache.MarketReader, p *cache.BorrowPosition) *Liquidable {
	out := &Liquidable{}
	out.Pos = p
	snap := mReader.GetSnapshot(p.MarketID)
	if snap == nil {
		out.SimErr = fmt.Errorf("snap nil")
		return out
	}

	params := c.MarketMap[p.MarketID]

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
		seizeAssets = snap.Stats.MaxUniSwappable
	}

	// 2. MinOut off-chain selon la liquidité du marché

	minOut := ComputeMinOut(seizeAssets, snap.Oracle.Price, snap.Oracle.Price)
	out.MinOut = minOut

	args := c.ToLiquidationArg(out, snap, params, minOut)
	data, err := lqtypes.EncodeLiquidateCalldata(args)

	if err != nil {
		out.SimErr = fmt.Errorf("encode: %w", err)
		return out
	}
	out.Args = args

	gasVal, err := c.dryRun(ctx, data)
	if err != nil {
		out.SimErr = fmt.Errorf("eth_call failed: %w", err)
		out.IsLiquidable = false
		return out
	}
	out.GasEstimate = gasVal

	out.SeizeAssets = seizeAssets
	c.log(fmt.Sprintf("seized asset %s with successfull simulation  %s", utils.FormatDecimals(out.SeizeAssets, int(params.CollateralTokenDecimals)), utils.FormatWAD(out.Pos.CachedHF)))
	out.SimulatedAt = time.Now()
	out.IsLiquidable = true

	return out
}
func (c *Consumer) LiquidateCall(ctx context.Context, args lqtypes.LiquidateArgs, gasEstimate uint64) error {
	calldata, err := lqtypes.EncodeLiquidateCalldata(args)
	if err != nil {
		c.log(fmt.Errorf("LiquidateCall: encode: %w", err).Error())
	}
	tx := onchain.TxParams{
		To:          &c.Config.Addresses.LiquidatorContract,
		Calldata:    calldata,
		GasEstimate: gasEstimate,
	}
	_, err = onchain.SendSignedTx(ctx, c.Conn, c.Config.Addresses.Wallet, c.Config.Signer, tx)
	return err
}
