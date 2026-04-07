package core

import (
	"context"
	"fmt"
	"maps"
	"math"
	"math/big"
	"os"
	"path"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/config"
	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/cex"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

/*


 */

// swap rpc provider si volatilité augmente a la baisse
func (c *Cache) Scan(conn *connector.Connector, cexFeed *cex.CoinbaseConnector) error {
	ctx, cancel := context.WithCancel(context.Background())
	logChannel := make(chan string, 50)
	defer cancel()
	errCh := make(chan error, 5)

	priceChan := cexFeed.PriceCh()
	// Cex Price routine updating cexCache
	go cexFeed.Run(ctx)
	// Event based position and markets updates
	go c.WatchPositionRoutine(ctx, conn, errCh, logChannel)
	// Onchain rpc pool to update markets
	go c.OnChainRefreshRoutine(ctx, conn, errCh, logChannel)
	// Loging Ethcalls per min
	go c.CountEthCallPerMinuteRoutine(ctx, conn, errCh, logChannel)

	go c.RebuildWatchListRoutine(ctx, conn, errCh, logChannel)
	go c.FireLiquidationRoutine(ctx, conn, errCh, logChannel)
	go c.LogMarketsRoutine(ctx, conn, errCh, logChannel)
	go c.WriteLogRoutine(ctx, errCh, logChannel)
	for {
		select {
		case priceUpdate := <-priceChan:
			c.CexCache.UpdateNonCorrelated(priceUpdate)

		case event := <-conn.PositionCh:
			c.ProcessEvents(event)
		}
	}
}

func (c *Cache) WatchPositionRoutine(ctx context.Context, conn *connector.Connector, errCh chan error, logChannel chan string) {
	conn.WatchPositions(ctx)
	errCh <- fmt.Errorf("WatchPositions exited unexpectedly")
}

func (c *Cache) OnChainRefreshRoutine(ctx context.Context, conn *connector.Connector, errCh chan error, logChannel chan string) {

	markets := slices.Collect(maps.Values(c.Config.Markets))

	for {
		refreshArr, timer, vol := c.CexCache.GetRefreshParams(markets, 2)

		logChannel <- fmt.Sprintf("%d ms", timer/1000)
		if vol > 2 {
			event := RebuildEvent{
				MarketIDs: morpho.NotCexOnlyIds(markets),
				Reason:    "high_vol_refresh_not_cex",
			}
			c.rebuildCh <- event

		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(timer):
			if err := c.OnChainRefresh(conn.ClientHTTP, refreshArr); err != nil {
				errCh <- err
			}
		}
	}
}

func (c *Cache) CountEthCallPerMinuteRoutine(ctx context.Context, conn *connector.Connector, errCh chan error, logChannel chan string) {
	utils.RunTicker(ctx, 1*time.Minute, errCh, func() error {
		lastMinuteCount := c.LastMinCallCount.Load()
		c.LastMinCallCount.Store(0)
		logChannel <- fmt.Sprintf("%d ETH_CALLS \n", lastMinuteCount)
		return nil
	})
}

func (c *Cache) RebuildWatchListRoutine(ctx context.Context, conn *connector.Connector, errCh chan error, logChannel chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-c.rebuildCh:

			c.rebuildWatchlist(conn.ClientHTTP, ctx, logChannel, event)
		}
	}
}

func (c *Cache) FireLiquidationRoutine(ctx context.Context, conn *connector.Connector, errCh chan error, logChannel chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case liquidable := <-c.liquidCh:
			_ = liquidable

		}
	}
}

func (c *Cache) LogMarketsRoutine(ctx context.Context, conn *connector.Connector, errCh chan error, logChannel chan string) {
	utils.RunTicker(ctx, 4*time.Minute, errCh, func() error {
		var sb strings.Builder

		for id, m := range c.PositionCache.m {
			m.Mu.RLock()
			if m.OraclePrice == nil || m.LLTV == nil ||
				m.MarketStats.TotalBorrowAssets == nil ||
				m.MarketStats.TotalBorrowShares == nil {
				m.Mu.RUnlock()
				continue
			}
			market := c.GetMorphoMarketFromId(id)
			oraclePrice := new(big.Int).Set(m.OraclePrice) // copie de valeur
			lltv := new(big.Int).Set(m.LLTV)
			positions, stats := c.GetMarketPropsValue(id)
			m.Mu.RUnlock()
			exposant := 36 + market.LoanTokenDecimals - market.CollateralTokenDecimals
			price := utils.BigIntToFloat(oraclePrice) / math.Pow10(int(exposant))
			borrowAssets := utils.BigIntWADToFloat(stats.TotalBorrowAssets)
			borrowShares := utils.BigIntWADToFloat(stats.TotalBorrowShares)

			fmt.Fprintf(&sb, "\n┌─ Market %s/%s\n", market.CollateralTokenStr, market.LoanTokenStr)
			fmt.Fprintf(&sb, "│  price:         %.6f\n", price)
			fmt.Fprintf(&sb, "│  borrow assets: %.2f\n", borrowAssets)
			fmt.Fprintf(&sb, "│  borrow shares: %.2f\n", borrowShares)

			type hfPos struct {
				hf  *big.Int
				pos BorrowPosition
			}
			var atrisk []hfPos
			for _, p := range positions {
				hf := p.HF(stats.TotalBorrowShares, stats.TotalBorrowAssets, oraclePrice, lltv)
				if hf.Cmp(utils.WAD1DOT1) < 0 || hf.Sign() != 0 {
					atrisk = append(atrisk, hfPos{hf, p})
				}
			}

			if len(atrisk) == 0 {
				fmt.Fprintf(&sb, "│  no positions at risk\n└─\n\n")
				continue
			}

			sort.Slice(atrisk, func(i, j int) bool {
				return atrisk[i].hf.Cmp(atrisk[j].hf) < 0 // croissant: les plus risqués en premier
			})

			n := min(10, len(atrisk))
			fmt.Fprintf(&sb, "│  at-risk positions (%d):\n", len(atrisk))
			for i, hp := range atrisk[:n] {
				hf := utils.BigIntWADToFloat(hp.hf)
				fmt.Fprintf(&sb, "│  [%2d] HF: %.4f  borrower: %s\n", i+1, hf, hp.pos.Address)
			}
			fmt.Fprintf(&sb, "└─\n\n")
		}

		logChannel <- sb.String()
		return nil
	})
}

func (c *Cache) WriteLogRoutine(ctx context.Context, errCh chan error, logChannel chan string) {
	var mu sync.Mutex
	logCache := make(map[int64]string)

	pathLog := path.Join("logs", c.Config.Chain.Name)
	file, _ := os.Create(pathLog)
	defer file.Close()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-logChannel:
				mu.Lock()
				logCache[time.Now().Unix()] = msg
				mu.Unlock()

			case err := <-errCh:
				mu.Lock()
				logCache[time.Now().Unix()] = err.Error()
				mu.Unlock()
			}
		}
	}()

	utils.RunTicker(ctx, 2*time.Minute, errCh, func() error {
		mu.Lock()
		defer mu.Unlock()
		if len(logCache) == 0 {
			return nil
		}

		for ts, msg := range logCache {
			_, _ = fmt.Fprintf(file, "[%s] %s\n", time.Unix(ts, 0).Format(time.RFC3339), msg)

		}

		// vide le cache
		clear(logCache)
		return nil
	})
}

func (c *Cache) OnChainRefreshEvent(client *w3.Client, toRefresh [][32]byte) error {
	err := c.OnChainRefresh(client, toRefresh)
	if err != nil {
		return err
	}
	event := RebuildEvent{
		MarketIDs: toRefresh,
		Reason:    "on_chain_refresh",
	}
	c.rebuildCh <- event
	return nil
}

func (c *Cache) OnChainRefresh(client *w3.Client, toRefresh [][32]byte) error {

	ctx := context.Background()
	var calls, marketCalls []w3types.RPCCaller
	var marketMap map[[32]byte]*MarketStats

	// calls based on tracker state
	arr := [][32]byte{}
	for k := range c.Config.Markets {
		arr = append(arr, k)
	}
	marketMap, marketCalls = c.OnChainCalls(arr)
	calls = append(calls, marketCalls...)

	if err := c.EthCallCtx(client, ctx, calls); err != nil {
		fmt.Println("❌ OnChainRefresh EthCall error:", err)
		return err
	}
	c.ApplyMarketStats(marketMap)

	return nil
}

func (c *Cache) OnChainCalls(toRefresh [][32]byte) (map[[32]byte]*MarketStats, []w3types.RPCCaller) {
	var calls []w3types.RPCCaller
	marketStates := make(map[[32]byte]*MarketStats, len(toRefresh))

	for _, id := range toRefresh {
		// market stats
		ms := MarketStats{
			TotalBorrowAssets: new(big.Int),
			TotalBorrowShares: new(big.Int),
			OraclePrice:       new(big.Int),
		}
		marketStates[id] = &ms
		calls = append(calls, eth.CallFunc(config.MorphoMain, config.MarketFunc, id).Returns(
			new(big.Int), new(big.Int),
			ms.TotalBorrowAssets, ms.TotalBorrowShares,
			new(big.Int), new(big.Int),
		))

		// oracle price — direct dans market
		calls = append(calls, eth.CallFunc(c.GetMorphoMarketFromId(id).Oracle, config.OraclePriceFunc).Returns(ms.OraclePrice))
	}

	return marketStates, calls
}

func (c *Cache) ApplyMarketStats(marketMap map[[32]byte]*MarketStats) {
	for id, m := range c.PositionCache.m {
		ms, ok := marketMap[id]
		if !ok {
			continue
		}
		m.Mu.Lock()
		m.MarketStats.TotalBorrowAssets = ms.TotalBorrowAssets
		m.MarketStats.TotalBorrowShares = ms.TotalBorrowShares
		m.MarketStats.OraclePrice = ms.OraclePrice
		m.MarketStats.LastUpdate = time.Now().Unix()
		m.Mu.Unlock()
	}
}
