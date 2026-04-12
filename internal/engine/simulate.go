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
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

type SimResult struct {
	Position     market.BorrowPosition
	MarketID     [32]byte
	RepayShares  *big.Int
	SeizeAssets  *big.Int
	GasEstimate  uint64
	EstProfit    *big.Int
	IsLiquidable bool
	SimErr       error
}

func GetCandidates(c state.MarketReader, simCache *SimCache) []*Liquidable {
	ids := c.Ids()

	type result struct {
		candidates []*Liquidable
	}

	resultsCh := make(chan result, len(ids))
	var wg sync.WaitGroup

	for _, id := range ids {
		snap := c.GetSnapshot(id)
		if snap == nil {
			continue
		}

		wg.Add(1)
		go func(snap *market.MarketSnapshot, id [32]byte) {
			defer wg.Done()
			var local []*Liquidable

			for _, pos := range snap.Positions {
				if simCache.Blacklisted(pos.Address) {
					continue
				}
				cp := pos
				hf := cp.HF(snap.Stats.TotalBorrowShares, snap.Stats.TotalBorrowAssets, snap.Oracle.Price, snap.LLTV)
				if hf == nil || hf.Sign() == 0 || hf.Cmp(utils.WAD) >= 0 {
					continue
				}
				local = append(local, &Liquidable{
					Pos:      &cp,
					MarketID: id,
					HF:       hf,
				})
			}
			resultsCh <- result{local}
		}(snap, id)
	}

	wg.Wait()
	close(resultsCh)

	var candidates []*Liquidable
	for r := range resultsCh {
		candidates = append(candidates, r.candidates...)
	}
	return candidates
}

func SimulateCandidates(conn *connector.Connector, c state.MarketReader, marketMap map[[32]byte]morpho.MarketParams, candidates []*Liquidable, logChan chan string, simCache *SimCache) []*Liquidable {
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
				logChan <- fmt.Sprintf("sim failed for %s: %v market %s", enriched.Pos.Address, enriched.SimErr, liq.MarketID)
				// logChan <- Position details for debug
				simCache.RecordFailure(enriched.Pos.Address)
				return
			}
   // call for Odos.PathId in seperate routine
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
