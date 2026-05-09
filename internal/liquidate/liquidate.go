package liquidate

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/connector"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/config"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

type EthCaller interface {
	EthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error
	FallBackEthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error
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
	Args         LiquidateArgs
}

type LiquidateArgs struct {
	MarketParams morpho.MarketContractParams
	Borrower     common.Address
	SeizedAssets *big.Int
	RepaidShares *big.Int
	SwapRouter   common.Address
	PoolFee      *big.Int
	MinOut       *big.Int
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
	Conn      EthCaller
	Cache     cache.MarketReader
	MarketMap map[[32]byte]morpho.MarketParams
	Config    config.Config
	Logger    chan string
	Ch        <-chan cache.BorrowPosition
}

type TxParams struct {
	To          *common.Address
	Calldata    []byte
	Value       *big.Int
	GasEstimate uint64
}

func (c *Consumer) log(msg string) {
	select {
	case c.Logger <- msg:
	default: // drop if full — never block the liquidation path
	}
}

// Pure math, zero RPC — unit testable
func (c *Consumer) ToLiquidationArg(l *Liquidable, snap *cache.MarketSnapshot, params morpho.MarketParams, minOut *big.Int) LiquidateArgs {
	return LiquidateArgs{
		*params.ToMarketContractParams(),
		l.Pos.Address,
		l.SeizeAssets,
		big.NewInt(0),
		c.Config.Addresses.UniSwapRouter,
		big.NewInt(int64(snap.Stats.SwapFee)),
		minOut,
	}
}

// ABI encode — testable isolément
func encodeLiquidateCalldata(args LiquidateArgs) ([]byte, error) {
	return config.FuncLiquidate.EncodeArgs(
		args.MarketParams,
		args.Borrower,
		args.SeizedAssets,
		args.RepaidShares,
		args.SwapRouter,
		args.PoolFee,
		args.MinOut,
	)
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

	if seizeAssets.Cmp(snap.Stats.MaxUniSwappable) > 0 {
		seizeAssets = snap.Stats.MaxUniSwappable
	}

	// 2. MinOut off-chain selon la liquidité du marché

	minOut := ComputeMinOut(seizeAssets, snap.Oracle.Price, snap.Oracle.Price)
	out.MinOut = minOut

	args := c.ToLiquidationArg(out, snap, params, minOut)
	data, err := encodeLiquidateCalldata(args)

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
func (c *Consumer) LiquidateCall(ctx context.Context, args LiquidateArgs, gasEstimate uint64) error {
	calldata, err := encodeLiquidateCalldata(args)
	if err != nil {
		c.log(fmt.Errorf("LiquidateCall: encode: %w", err).Error())
	}

	_, err = c.SendSignedTx(ctx, TxParams{
  // c.Config.Address.LiquidatorAddr
		To:          &config.BaseLiqu,
		Calldata:    calldata,
		GasEstimate: gasEstimate,
	})
	return err
}

func (c *Consumer) SendSignedTx(ctx context.Context, params TxParams) (common.Hash, error) {
	var nonce uint64
	var gasPrice *big.Int

	if err := c.Conn.FallBackEthCallCtx(ctx, []w3types.RPCCaller{
		eth.Nonce(c.Config.Addresses.Wallet, nil).Returns(&nonce),
		eth.GasPrice().Returns(&gasPrice),
	}); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: fetch params: %w", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		To:        params.To,
		Data:      params.Calldata,
		Value:     params.Value,
		Gas:       params.GasEstimate * 12 / 10,
		GasTipCap: big.NewInt(1e9),
		GasFeeCap: new(big.Int).Add(gasPrice, big.NewInt(1e9)),
	})

	signedTx, err := c.Config.Signer.Sign(tx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: sign: %w", err)
	}

	var receipt common.Hash
	if err := c.Conn.EthCallCtx(ctx, []w3types.RPCCaller{
		eth.SendTx(signedTx).Returns(&receipt),
	}); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: send: %w", err)
	}

	log.Printf("[tx] sent: %s", receipt.Hex())
	return receipt, nil
}
