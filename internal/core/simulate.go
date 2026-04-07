package core

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/config"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/cex"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

/* Loop over liquidable and test liquidate call with precompute values */
func (c *Cache) simulateCandidates(client *w3.Client, ctx context.Context, candidates []*Liquidable) []*Liquidable {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []*Liquidable
	sem := make(chan struct{}, 5) // max 5 appels RPC simultanés
	for _, liq := range candidates {
		wg.Add(1)
		go func(liq *Liquidable) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			enriched := c.SimulatePreComputeTx(client, ctx, liq)
			if enriched.SimErr != nil || enriched.EstProfit.Sign() <= 0 || !enriched.IsLiquidable {
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

func (c *Cache) rebuildWatchlist(client *w3.Client, ctx context.Context, logChan chan string, event RebuildEvent) {
	var flat []*Liquidable
	var wg sync.WaitGroup
	var mu sync.Mutex

	for mId := range c.PositionCache.m {
		wg.Add(1)
		go func(mId [32]byte) {
			defer wg.Done()
			positions, stats := c.GetMarketProps(mId)
			var local []*Liquidable
			for _, p := range positions {
				market := c.GetMorphoMarketFromId(mId)
				var cexPrice *big.Int
				if market.CexOnly {
					cexPrice = cex.GetCollateralPriceInLoan(c.CexCache, &market)
				}
				if cexPrice == nil {
					cexPrice = stats.OraclePrice
				}
				hf := p.HF(stats.TotalBorrowShares, stats.TotalBorrowAssets, cexPrice, stats.LLTV)
				isNotInTargetZone := hf.Cmp(utils.WAD1DOT01) >= 0
				if market.Correlated {
					isNotInTargetZone = hf.Cmp(utils.WAD1DOT0005) >= 0
				}
				if hf == nil || hf.Sign() == 0 {
					continue
				}
				if isNotInTargetZone {
					continue
				}
				if p.SimulationCount > 8 {
					continue
				}
				local = append(local, &Liquidable{HF: hf, Pos: p, MarketID: mId})
			}
			if len(local) == 0 {
				return
			}
			logChan <- fmt.Sprintf("%d liquidable", len(local))
			mu.Lock()
			flat = append(flat, local...)
			mu.Unlock()
		}(mId)
	}
	wg.Wait()

	enriched := c.simulateCandidates(client, ctx, flat)
	if len(enriched) > 0 {
		for _, l := range enriched {
			c.liquidCh <- *l
			logChan <- fmt.Sprintf("liquidable pos %v", l)
		}
	}

}

/* Get Liquidations params by calculating and calling eth_call to test for revert */
func (c *Cache) SimulatePreComputeTx(client *w3.Client, ctx context.Context, liq *Liquidable) *Liquidable {
	out := *liq
	params := c.Config.Markets[liq.MarketID]
	market := c.PositionCache.m[liq.MarketID]
	market.Mu.RLock()
	stats := market.MarketStats
	market.Mu.RUnlock()

	// 1. Math pure — pas de RPC
	repayShares, seizeAssets := morpho.ComputeLiquidationAmounts(liq.Pos.BorrowShares, stats.TotalBorrowAssets, stats.TotalBorrowShares, params)
	out.RepayShares = repayShares
	out.SeizeAssets = seizeAssets

	// 2. Dry-run eth_call
	gasEst, err := c.simulateLiquidationCall(client, ctx, params, liq.Pos, repayShares)
	if err != nil {
		log.Printf("simulation failed: %s \n", err.Error())
		out.SimErr = fmt.Errorf("simulation failed: %w", err)
		return &out
	}
	out.GasEstimate = gasEst

	// 3. Profit net
	out.EstProfit = morpho.EstimateProfit(seizeAssets, repayShares, gasEst)
	out.SimulatedAt = time.Now()
	out.IsLiquidable = true

	return &out
}

/* ETH_CALL to check for revert */
func (c *Cache) simulateLiquidationCall(
	client *w3.Client,
	ctx context.Context,
	params morpho.MarketParams,
	pos *BorrowPosition,
	repayShares *big.Int,
) (uint64, error) {

	// 1. Encode calldata via FuncLiquidate
	data, err := config.FuncLiquidate.EncodeArgs(
		params.ToMarketContractParams(),
		pos.Address,
		repayShares,
		c.Config.Chain.UniswapRouterAddress, // adapt to chainId
		params.PoolFee,                      // adapt to chainId
	)
	if err != nil {
		return 0, fmt.Errorf("encode liquidate: %w", err)
	}
	msg := w3types.Message{
		From:  c.Config.Chain.WalletAddress,      // adapt to chainId
		To:    &c.Config.Chain.LiquidatorAddress, // adapt to chainId
		Input: data,
	}

	var result []byte
	var gasVal uint64
	var callers []w3types.RPCCaller
	callers = append(callers, eth.Call(&msg, nil, nil).Returns(&result))
	callers = append(callers, eth.EstimateGas(&msg, nil).Returns(&gasVal))
	pos.SimulationCount += 1
	if err := c.EthCallCtx(client, ctx, callers); err != nil {
		return 0, fmt.Errorf("eth_call failed: %w", err)
	}
	return gasVal, nil
}

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
