package engine

import (
	"math/big"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
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
func rebuildWatchlist(c *core.Cache, client *w3.Client, ctx context.Context, logChan chan string) {
	var flat []*Liquidable
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, mId := range c.Markets.Ids() {
		wg.Add(1)
		go func(mId [32]byte) {
			defer wg.Done()
			snap := c.Markets.GetSnapshot(mId)
			stats := snap.Stats
			oracle := snap.Oracle
			var local []*Liquidable
			for _, p := range snap.Positions {
				hf := p.HF(stats.TotalBorrowShares, stats.TotalBorrowAssets, oracle.Price, snap.LLTV)
				isNotInTargetZone := hf.Cmp(utils.WAD) >= 0
				if hf == nil || hf.Sign() == 0 || isNotInTargetZone {
					continue
				}
				local = append(local, &Liquidable{HF: hf, Pos: &p, MarketID: mId})
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
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	enriched := c.simulateCandidates(client, ctx, flat, logChan)
	for _, l := range enriched {
		select {
		case c.liquidCh <- *l:
		case <-ctx.Done():
			log.Println("liquidCh timeout, positions ignorées")
			return
		}
	}

}

func simulateCandidates(c *core.Cache, client *w3.Client, ctx context.Context, candidates []*Liquidable, logChan chan string) []*Liquidable {
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
			enriched := SimulatePreComputeTx(c, client, ctx, liq)
			if enriched.SimErr != nil || enriched.EstProfit.Sign() <= 0 || !enriched.IsLiquidable {
				logChan <- fmt.Sprintf("Err in liq simulation: %s for borrower %s with %d repaid shares on %s", enriched.SimErr, enriched.Pos.Address, enriched.RepayShares, c.GetMorphoMarketFromId(liq.MarketID).CollateralTokenStr)
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

func SimulatePreComputeTx(c *core.Cache, client *w3.Client, ctx context.Context, liq *Liquidable) *Liquidable {
	out := *liq

	market := c.Markets.GetSnapshot(liq.MarketID)
	stats := market.Stats

	// 1. Math pure — pas de RPC
	repayShares, seizeAssets := morpho.ComputeLiquidationAmounts(liq.Pos.BorrowShares, stats.TotalBorrowAssets, stats.TotalBorrowShares, params)
	out.RepayShares = repayShares
	out.SeizeAssets = seizeAssets

	// 2. Dry-run eth_call
	gasEst, err := c.simulateLiquidationCall(client, ctx, params, liq.Pos, repayShares)
	if err != nil {
		log.Printf("simulation failed: %s for borrower %s \n", err.Error(), out.Pos.Address.String())
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

func simulateLiquidationCall(
	c *core.Cache,
	client *w3.Client,
	ctx context.Context,
	params morpho.MarketParams,
	pos *position.BorrowPosition,
	repayShares *big.Int,
) (uint64, error) {

	// 1. Encode calldata via FuncLiquidate
	data, err := config.FuncLiquidate.EncodeArgs(
		params.ToMarketContractParams(),
		pos.Address,
		big.NewInt(0),
		repayShares,
		c.Config.Chain.UniswapRouterAddress, // adapt to chainId
		big.NewInt(int64(params.PoolFee)),   // adapt to chainId
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

*/
