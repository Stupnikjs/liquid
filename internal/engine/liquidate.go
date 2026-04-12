package engine

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
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
	MarketParams  morpho.MarketContractParams
	Borrower      common.Address
	SeizedAssets  *big.Int
	RepaidShares  *big.Int
	OdosPathId    string
	OdosAmountOut *big.Int
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
