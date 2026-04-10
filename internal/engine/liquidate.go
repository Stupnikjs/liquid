package engine

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
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

/*
func (c *Cache) LiquidateCall(

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
			From:  c.Config.Chain.WalletAddress,      // adapt to chainId
			To:    &c.Config.Chain.LiquidatorAddress, // adapt to chainId
			Input: calldata,
		}

		if err := client.CallCtx(ctx,
			eth.Nonce(c.Config.Chain.WalletAddress, nil).Returns(&nonce),
			eth.GasPrice().Returns(&gasPrice),
			eth.EstimateGas(&msg, nil).Returns(&gasEst),
		); err != nil {
			return fmt.Errorf("LiquidateCall: fetch tx params: %w", err)
		}

		tx := types.NewTx(&types.DynamicFeeTx{
			Nonce:     nonce,
			To:        &c.Config.Chain.LiquidatorAddress,
			Data:      calldata,
			Gas:       gasEst * 12 / 10, // +20% marge
			GasTipCap: big.NewInt(1e9),  // 1 gwei tip
			GasFeeCap: new(big.Int).Add(gasPrice, big.NewInt(1e9)),
		})

		// Sign
		signedTx, err := c.Config.Chain.Signer.Sign(tx)

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
*/
func SimulateCandidates(conn *connector.Connector, c state.MarketReader, marketMap map[[32]byte]morpho.MarketParams, candidates []*Liquidable, logChan chan string) []*Liquidable {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []*Liquidable
	sem := make(chan struct{}, 5)

	for _, liq := range candidates {
		wg.Add(1)
		go func(liq *Liquidable) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			enriched := SimulatePreComputeTx(conn, c, marketMap, liq)
			if enriched.SimErr != nil || enriched.EstProfit.Sign() <= 0 || !enriched.IsLiquidable {
				logChan <- fmt.Sprintf("sim failed for %s: %v", enriched.Pos.Address, enriched.SimErr)
				return
			}
			mu.Lock()
			results = append(results, enriched)
			mu.Unlock()
		}(liq)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		return results[i].EstProfit.Cmp(results[j].EstProfit) > 0
	})
	return results
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
		From:  config.BaseWalletAddr,      // change for multichain
		To:    &config.BaseLiquidatorAddr, //
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
