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
	"github.com/Stupnikjs/morpho-sepolia/pkg/swap"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

func GetCandidates(c state.MarketReader, simCache *SimCache) []*Liquidable {
	ids := c.Ids()
	resultsCh := make(chan []*Liquidable, len(ids))
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
				hf := cp.HF(
					snap.Stats.TotalBorrowShares,
					snap.Stats.TotalBorrowAssets,
					snap.Oracle.Price,
					snap.LLTV,
				)
				if hf == nil || hf.Sign() == 0 || hf.Cmp(utils.WAD) >= 0 {
					continue
				}
				local = append(local, &Liquidable{
					Pos:      &cp,
					MarketID: id,
					HF:       hf,
				})
			}
			resultsCh <- local
		}(snap, id)
	}

	wg.Wait()
	close(resultsCh)

	var candidates []*Liquidable
	for r := range resultsCh {
		candidates = append(candidates, r...)
	}
	return candidates
}

func SimulateCandidates(
	conn *connector.Connector,
	c state.MarketReader,
	marketMap map[[32]byte]morpho.MarketParams,
	candidates []*Liquidable,
	logChan chan string,
	simCache *SimCache,
) []*Liquidable {
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
				logChan <- fmt.Sprintf("sim failed for %s: %v market %x", enriched.Pos.Address, enriched.SimErr, liq.MarketID)
				simCache.RecordFailure(enriched.Pos.Address)
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

func SimulatePreComputeTx(
	conn *connector.Connector,
	c state.MarketReader,
	marketMap map[[32]byte]morpho.MarketParams,
	liq *Liquidable,
) *Liquidable {
	out := *liq
	snap := c.GetSnapshot(liq.MarketID)
	if snap == nil {
		out.SimErr = fmt.Errorf("snap nil")
		return &out
	}

	params := marketMap[liq.MarketID]

	repayShares, seizeAssets := morpho.ComputeLiquidationAmounts(
		liq.Pos.BorrowShares,
		snap.Stats.TotalBorrowAssets,
		snap.Stats.TotalBorrowShares,
		snap.LLTV,
	)
	out.RepayShares = repayShares
	out.SeizeAssets = seizeAssets

	// 1. Quote Odos
	quote, err := swap.Quote(params, seizeAssets)
	if err != nil {
		out.SimErr = fmt.Errorf("odos quote: %w", err)
		return &out
	}

	// 2. Assemble pour avoir le vrai calldata
	odosCalldata, err := swap.AssembleOdos(quote.PathId, config.BaseLiquidatorAddrV2)
	if err != nil {
		out.SimErr = fmt.Errorf("odos assemble: %w", err)
		return &out
	}

	// 3. Encode l'appel liquidate avec le calldata Odos
	data, err := config.FuncLiquidate.EncodeArgs(
		params.ToMarketContractParams(),
		liq.Pos.Address,
		big.NewInt(0),
		repayShares,
		config.OdosRouterAddr,
		odosCalldata,
	)
	if err != nil {
		out.SimErr = fmt.Errorf("encode: %w", err)
		return &out
	}

	msg := w3types.Message{
		From:  config.BaseWalletAddr,
		To:    &config.BaseLiquidatorAddrV2,
		Input: data,
	}

	// 4. eth_call + estimateGas en batch
	var gasVal uint64
	var callResult []byte
	if err := conn.EthCallCtx(context.Background(), []w3types.RPCCaller{
		eth.Call(&msg, nil, nil).Returns(&callResult),
		eth.EstimateGas(&msg, nil).Returns(&gasVal),
	}); err != nil {
		out.SimErr = fmt.Errorf("eth_call failed: %w", err)
		return &out
	}

	out.GasEstimate = gasVal
	out.EstProfit = morpho.EstimateProfit(seizeAssets, repayShares, gasVal)
	out.SimulatedAt = time.Now()
	out.IsLiquidable = true
	out.OdosCallData = odosCalldata
	return &out
}

func (e *Engine) ExecuteLiquidation(
	ctx context.Context,
	liq *Liquidable,
	marketMap map[[32]byte]morpho.MarketParams,
) error {
	params := marketMap[liq.MarketID]

	// re-Quote + re-Assemble — calldata frais garanti
	quote, err := swap.Quote(params, liq.SeizeAssets)
	if err != nil {
		return fmt.Errorf("re-quote: %w", err)
	}

	odosCalldata, err := swap.AssembleOdos(quote.PathId, config.BaseLiquidatorAddrV2)
	if err != nil {
		return fmt.Errorf("re-assemble: %w", err)
	}

	data, err := config.FuncLiquidate.EncodeArgs(
		params.ToMarketContractParams(),
		liq.Pos.Address,
		big.NewInt(0),
		liq.RepayShares,
		config.OdosRouterAddr,
		odosCalldata,
	)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	_, err = e.SendSignedTx(ctx, TxParams{
		To:       &config.BaseLiquidatorAddrV2,
		Calldata: data,
	})
	return err
}
