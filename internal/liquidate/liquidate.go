package liquidate

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/cache"
	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

var (
	LiquidatorAddr = common.HexToAddress("0xYOUR_LIQUIDATOR_CONTRACT")
	SwapRouterAddr = common.HexToAddress("0x2626664c2603336E57B271c5C0b26F421741e481")
)

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

func NewConsumer(conn *connector.Connector, marketReader state.MarketReader, marketMap map[[32]byte]morpho.MarketParams, signer *config.Signer, logger chan string, ch <-chan cache.BorrowPosition) *Consumer {
	return &Consumer{
		Conn:      conn,
		Cache:     marketReader,
		MarketMap: marketMap,
		Signer:    signer,
		Logger:    logger,
		Ch:        ch,
	}
}

type Consumer struct {
	Conn      *connector.Connector
	Cache     state.MarketReader
	MarketMap map[[32]byte]morpho.MarketParams
	Signer    *config.Signer
	Logger    chan string
	Ch        <-chan cache.BorrowPosition
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

func (c *Consumer) liquidateWrapper(ctx context.Context, mReader state.MarketReader, p *cache.BorrowPosition) {
	result := c.SimulateAndPreComputeTx(ctx, mReader, p)
	if result.SimErr != nil {
		c.Logger <- fmt.Sprintf("[liq] simulation failed for %s: %v", p.Address, result.SimErr)
		return
	}
	if !result.IsLiquidable {
		c.Logger <- "[liq] not profitable"
		return
	}
	market := c.MarketMap[p.MarketID]
	c.Logger <- fmt.Sprintf("[liq] sending tx for %s seized=%s market %s ", p.Address, utils.FormatDecimals(result.SeizeAssets, int(market.CollateralTokenDecimals)), market.GetPair())

	err := c.LiquidateCall(ctx, result.Args)
	if err != nil {
		c.Logger <- fmt.Sprintf("[liq] tx failed for %s: %v", p.Address, err)
		return
	}
	c.Logger <- fmt.Sprintf("[liq] ✓ liquidated pair%s marketid:%s borrower:%s", market.GetPair(), hexutil.Encode(market.ID[:]), p.Address)
}

func (c *Consumer) SendSignedTx(ctx context.Context, params TxParams) (common.Hash, error) {
	var nonce uint64
	var gasPrice *big.Int
	var gasEst uint64

	msg := w3types.Message{
		From:  config.BaseWalletAddr,
		To:    params.To,
		Input: params.Calldata,
		Value: params.Value,
	}

	if err := c.Conn.ClientHTTP.CallCtx(ctx,
		eth.Nonce(config.BaseWalletAddr, nil).Returns(&nonce),
		eth.GasPrice().Returns(&gasPrice),
		eth.EstimateGas(&msg, nil).Returns(&gasEst),
	); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: fetch params: %w", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		To:        params.To,
		Data:      params.Calldata,
		Value:     params.Value,
		Gas:       gasEst * 12 / 10,
		GasTipCap: big.NewInt(1e9),
		GasFeeCap: new(big.Int).Add(gasPrice, big.NewInt(1e9)),
	})

	signedTx, err := c.Signer.Sign(tx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: sign: %w", err)
	}

	var receipt common.Hash
	if err := c.Conn.ClientHTTP.CallCtx(ctx, eth.SendTx(signedTx).Returns(&receipt)); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: send: %w", err)
	}

	log.Printf("[tx] sent: %s", receipt.Hex())
	return receipt, nil
}

type TxParams struct {
	To       *common.Address
	Calldata []byte
	Value    *big.Int
}

func (c *Consumer) LiquidateCall(ctx context.Context, args LiquidateArgs) error {
	calldata, err := config.FuncLiquidate.EncodeArgs(
		args.MarketParams,
		args.Borrower,
		args.SeizedAssets,
		args.RepaidShares,
		args.SwapRouter,
		args.PoolFee,
		args.MinOut,
	)
	if err != nil {
		return fmt.Errorf("LiquidateCall: encode: %w", err)
	}

	_, err = c.SendSignedTx(ctx, TxParams{
		To:       &config.BaseLiquidatorAddr,
		Calldata: calldata,
	})
	return err
}

func ComputeMinOut(seizedAssets, collateralPrice, loanPrice *big.Int) *big.Int {
	valueInLoan := new(big.Int).Mul(seizedAssets, collateralPrice)
	valueInLoan.Div(valueInLoan, loanPrice)
	return valueInLoan
}

func (c *Consumer) SimulateAndPreComputeTx(ctx context.Context, mReader state.MarketReader, p *cache.BorrowPosition) *Liquidable {
	out := &Liquidable{}
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

	args := LiquidateArgs{
		*params.ToMarketContractParams(),
		p.Address,
		seizeAssets,
		big.NewInt(0),
		config.BaseUniswapV3Router,
		big.NewInt(int64(snap.Stats.SwapFee)),
		minOut,
	}
	// 3. Dry-run eth_call + EstimateGas en batch
	data, err := config.FuncLiquidate.EncodeArgs(
		args.MarketParams,
		args.Borrower,
		args.SeizedAssets,
		args.RepaidShares,
		args.SwapRouter,
		args.PoolFee,
		args.MinOut)

	if err != nil {
		out.SimErr = fmt.Errorf("encode: %w", err)
		return out
	}
	out.Args = args

	msg := w3types.Message{
		From:  config.BaseWalletAddr,
		To:    &config.BaseLiquidatorAddr,
		Input: data,
	}

	var gasVal uint64
	var callResult []byte
	if err := c.Conn.EthCallCtx(context.Background(), []w3types.RPCCaller{
		eth.Call(&msg, nil, nil).Returns(&callResult),
		eth.EstimateGas(&msg, nil).Returns(&gasVal),
	}); err != nil {
		out.SimErr = fmt.Errorf("eth_call failed: %w", err)
		out.IsLiquidable = false
		return out
	}

	// 4. Profit net
	out.GasEstimate = gasVal
	out.SeizeAssets = seizeAssets
	c.Logger <- fmt.Sprintf("seized asset %s with successfull simulation  %s", utils.FormatDecimals(out.SeizeAssets, int(params.CollateralTokenDecimals)), utils.FormatWAD(out.Pos.CachedHF))
	out.SimulatedAt = time.Now()
	out.IsLiquidable = true

	return out
}
