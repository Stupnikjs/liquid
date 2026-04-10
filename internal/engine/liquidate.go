package engine

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

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

func LiquidateCall(
	signer *morpho.Signer,
	m state.MarketReader,
	client *w3.Client,
	ctx context.Context,
	marketParams morpho.MarketContractParams,
	borrower common.Address,
	seizedAssets *big.Int,
	repaidShares *big.Int,
	swapRouter common.Address,
	poolFee *big.Int,

) error {
	calldata, err := config.FuncLiquidate.EncodeArgs(
		marketParams,
		borrower,
		seizedAssets,
		repaidShares,
		swapRouter,
		poolFee,
	)
	if err != nil {
		return fmt.Errorf("LiquidateCall: encode args: %w", err)
	}
	var nonce uint64
	var gasPrice *big.Int
	var gasEst uint64

	msg := w3types.Message{
		From:  config.BaseWalletAddr,      // adapt to chainId
		To:    &config.BaseLiquidatorAddr, // adapt to chainId
		Input: calldata,
	}

	if err := client.CallCtx(ctx,
		eth.Nonce(config.BaseWalletAddr, nil).Returns(&nonce),
		eth.GasPrice().Returns(&gasPrice),
		eth.EstimateGas(&msg, nil).Returns(&gasEst),
	); err != nil {
		return fmt.Errorf("LiquidateCall: fetch tx params: %w", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		To:        &config.BaseLiquidatorAddr,
		Data:      calldata,
		Gas:       gasEst * 12 / 10, // +20% marge
		GasTipCap: big.NewInt(1e9),  // 1 gwei tip
		GasFeeCap: new(big.Int).Add(gasPrice, big.NewInt(1e9)),
	})

	// Sign

	signedTx, err := signer.Sign(tx)

	if err != nil {
		return fmt.Errorf("LiquidateCall: sign tx: %w", err)
	}

	var receipt common.Hash
	if err := client.CallCtx(ctx, eth.SendTx(signedTx).Returns(&receipt)); err != nil {
		return fmt.Errorf("LiquidateCall: send tx: %w", err)
	}

	log.Printf("[liquidate] tx sent: %s", receipt.Hex())
	return nil
}
