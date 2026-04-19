package liquidate

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

var (
	LiquidatorAddr = common.HexToAddress("0xYOUR_LIQUIDATOR_CONTRACT")
	SwapRouterAddr = common.HexToAddress("0x2626664c2603336E57B271c5C0b26F421741e481")

	FuncLiquidate = w3.MustNewFunc(
		`liquidate(
			(address loanToken, address collateralToken, address oracle, address irm, uint256 lltv) marketParams,
			address borrower,
			uint256 seizedAssets,
			uint256 repaidShares,
			address swapRouter,
			uint24 poolFee
		)`,
		``,
	)
)

type Liquidable struct {
	Pos          *market.BorrowPosition
	MarketID     [32]byte
	HF           *big.Int
	RepayShares  *big.Int
	SeizeAssets  *big.Int
	EstProfit    *big.Int
	GasEstimate  uint64
	SimulatedAt  time.Time
	SimErr       error
	IsLiquidable bool
}

type LiquidationEngine struct {
	RebuildCh   chan bool
	LiquidateCh chan *Liquidable
}

func New() *LiquidationEngine {
	return &LiquidationEngine{
		RebuildCh:   make(chan bool, 1),
		LiquidateCh: make(chan *Liquidable, 1),
	}
}

type LiquidateArgs struct {
	MarketParams morpho.MarketContractParams
	Borrower     common.Address
	SeizedAssets *big.Int
	RepaidShares *big.Int
	SwapRouter   common.Address
	PoolFee      *big.Int
}

func SendSignedTx(signer *config.Signer, client *w3.Client, ctx context.Context, params TxParams) (common.Hash, error) {
	var nonce uint64
	var gasPrice *big.Int
	var gasEst uint64

	msg := w3types.Message{
		From:  config.BaseWalletAddr,
		To:    params.To,
		Input: params.Calldata,
		Value: params.Value,
	}

	if err := client.CallCtx(ctx,
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

	signedTx, err := signer.Sign(tx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: sign: %w", err)
	}

	var receipt common.Hash
	if err := client.CallCtx(ctx, eth.SendTx(signedTx).Returns(&receipt)); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: send: %w", err)
	}

	log.Printf("[tx] sent: %s", receipt.Hex())
	return receipt, nil
}

type TxParams struct {
	To       *common.Address
	Calldata []byte
	Value    *big.Int // nil = 0
}

func LiquidateCall(signer *config.Signer, client *w3.Client, ctx context.Context, args LiquidateArgs) error {
	calldata, err := config.FuncLiquidate.EncodeArgs(args)
	if err != nil {
		return fmt.Errorf("LiquidateCall: encode: %w", err)
	}

	_, err = SendSignedTx(signer, client, ctx, TxParams{
		To:       &config.BaseLiquidatorAddrV2,
		Calldata: calldata,
	})
	return err
}

func SimulatePreComputeTx(conn *connector.Connector, c state.MarketReader, marketMap map[[32]byte]morpho.MarketParams, liq *Liquidable) *Liquidable {
	out := *liq
	snap := c.GetSnapshot(liq.MarketID)
	if snap == nil {
		out.SimErr = fmt.Errorf("snap nil")
		return &out
	}

	params := marketMap[liq.MarketID]

	// 1. Math pure — pas de RPC
	repayShares, seizeAssets := morpho.ComputeLiquidationAmounts(
		liq.Pos.BorrowShares,
		snap.Stats.TotalBorrowAssets,
		snap.Stats.TotalBorrowShares,
		snap.LLTV,
	)
	out.RepayShares = repayShares
	out.SeizeAssets = seizeAssets

	// 2. Dry-run eth_call + EstimateGas en batch
	data, err := config.FuncLiquidate.EncodeArgs(
		params.ToMarketContractParams(),
		liq.Pos.Address,
		big.NewInt(0),
		repayShares,
		config.BaseUniswapV3Router, // change for multichain
		big.NewInt(int64(params.PoolFee)),
	)
	if err != nil {
		out.SimErr = fmt.Errorf("encode: %w", err)
		return &out
	}

	msg := w3types.Message{
		From:  config.BaseWalletAddr,        // change for multichain
		To:    &config.BaseLiquidatorAddrV2, //
		Input: data,
	}

	var gasVal uint64
	var callResult []byte
	if err := conn.EthCallCtx(context.Background(), []w3types.RPCCaller{
		eth.Call(&msg, nil, nil).Returns(&callResult),
		eth.EstimateGas(&msg, nil).Returns(&gasVal),
	}); err != nil {
		out.SimErr = fmt.Errorf("eth_call failed: %w", err)
		return &out
	}

	// 3. Profit net
	out.GasEstimate = gasVal
	out.EstProfit = morpho.EstimateProfit(seizeAssets, repayShares, gasVal)
	out.SimulatedAt = time.Now()
	out.IsLiquidable = true

	return &out
}
