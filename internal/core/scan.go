package core

import (
	"context"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
)

type Runner struct {
	Cache *Cache
}

func (r *Runner) Scan(conn *connector.Connector) error {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	go r.WatchPositionRoutine(ctx, conn)
	// Onchain rpc pool to update markets
	go r.OnChainRefreshRoutine(ctx, conn)
	// Loging Ethcalls per min
	for {
		event := <-conn.PositionCh
		r.Cache.ProcessEvents(event)

	}
}

func (r *Runner) WatchPositionRoutine(ctx context.Context, conn *connector.Connector) {
	conn.WatchPositions(ctx)
}

func (r *Runner) OnChainRefreshRoutine(ctx context.Context, conn *connector.Connector) {
	errCh := make(chan error)
	utils.RunTicker(ctx, 2*time.Minute, errCh, func() error {
		return r.Cache.OnChainRefresh(conn.ClientHTTP)
	})
}

/*
func (c *Cache) RebuildWatchListRoutine(ctx context.Context, conn *connector.Connector, errCh chan error, logChannel chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-c.rebuildCh:
			_ = event
			c.rebuildWatchlist(conn.ClientHTTP, ctx, logChannel)
		}
	}
}

func (c *Cache) FireLiquidationRoutine(ctx context.Context, conn *connector.Connector, errCh chan error, logChannel chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case liquidable := <-c.liquidCh:
			market := c.GetMorphoMarketFromId(liquidable.MarketID)
			c.GetMorphoMarketFromId(liquidable.MarketID)
			c.LiquidateCall(
				conn.ClientHTTP,
				ctx,
				*market.ToMarketContractParams(),
				liquidable.Pos.Address,
				liquidable.SeizeAssets,
				liquidable.RepayShares,
				c.Config.Chain.UniswapRouterAddress,
				big.NewInt(int64(market.PoolFee)),
			)

		}
	}
}

func (c *Cache) MarketsRoutine(ctx context.Context, conn *connector.Connector, errCh chan error, logChannel chan string) {
	utils.RunTicker(ctx, 2*time.Minute, errCh, func() error {
		var sb strings.Builder

		for id, m := range c.PositionCache.m {
			market := c.GetMorphoMarketFromId(id)
			m.Mu.RLock()
			if m.OraclePrice == nil || m.LLTV == nil ||
				m.MarketStats.TotalBorrowAssets == nil ||
				m.MarketStats.TotalBorrowShares == nil ||
				m.Canceled {
				m.Mu.RUnlock()
				continue
			}

			// Tout copier sous le même lock
			oraclePrice := new(big.Int).Set(m.OraclePrice)
			lltv := new(big.Int).Set(m.LLTV)
			totalBorrowAssets := new(big.Int).Set(m.MarketStats.TotalBorrowAssets)
			totalBorrowShares := new(big.Int).Set(m.MarketStats.TotalBorrowShares)

			// Copier les positions sous le même lock
			positions := make([]*BorrowPosition, 0, len(m.Positions))
			for _, p := range m.Positions {
				cp := *p
				positions = append(positions, &cp)
			}

			m.Mu.RUnlock()
			if len(positions) < 5 {
				c.CancelMarket(id)
			}
			exposant := 36 + market.LoanTokenDecimals - market.CollateralTokenDecimals
			price := utils.BigIntToFloat(oraclePrice) / math.Pow10(int(exposant))
			borrowAssets := utils.BigIntToFloat(totalBorrowAssets) / math.Pow10(int(market.LoanTokenDecimals))
			borrowShares := utils.BigIntWADToFloat(totalBorrowShares)

			fmt.Fprintf(&sb, "\n┌─ Market %s/%s\n", market.CollateralTokenStr, market.LoanTokenStr)
			fmt.Fprintf(&sb, "│  price:         %.6f\n", price)
			fmt.Fprintf(&sb, "│  borrow assets: %.2f\n", borrowAssets)
			fmt.Fprintf(&sb, "│  borrow shares: %.2f\n", borrowShares)
			fmt.Fprintf(&sb, "│  positions less than 10pct from liquidation: %d\n", len(positions))
			type hfPos struct {
				hf  *big.Int
				pos BorrowPosition
			}
			var atrisk []hfPos

			for _, p := range positions {
				hf := p.HF(totalBorrowShares, totalBorrowAssets, oraclePrice, lltv)
				if hf.Cmp(utils.WAD1DOT1) < 0 && hf.Sign() != 0 {
					atrisk = append(atrisk, hfPos{hf, *p})
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
			fmt.Fprintf(&sb, "│  10 most at-risk positions (%d):\n", len(atrisk))
			for i, hp := range atrisk[:n] {
				hf := utils.BigIntWADToFloat(hp.hf)
				collAssets := utils.BigIntToFloat(hp.pos.CollateralAssets) / math.Pow10(int(float64(market.CollateralTokenDecimals)))
				fmt.Fprintf(&sb, "│  [%2d] HF: %.4f  borrower: %s borrow_assets: %.4f \n", i+1, hf, hp.pos.Address, collAssets)
			}
			fmt.Fprintf(&sb, "└─\n\n")
		}

		logChannel <- sb.String()
		return nil
	})
}

*/
