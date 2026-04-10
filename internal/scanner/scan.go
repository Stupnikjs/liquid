package scanner

import (
	"context"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/engine"
	"github.com/Stupnikjs/morpho-sepolia/internal/logging"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
)

type Runner struct {
	Cache  *Cache
	Engine *engine.LiquidationEngine
	Conn   *connector.Connector
	Logger chan string
}

func NewRunner(conn *connector.Connector, cache *Cache) *Runner {
	return &Runner{
		Cache:  cache,
		Engine: engine.New(),
		Conn:   conn,
		Logger: logging.NewLogger(context.Background(), "logg"),
	}
}

func (r *Runner) Run(ctx context.Context) {
	r.ApiCallRoutine(ctx)
	go r.WatchPositionRoutine(ctx)
	// Onchain rpc pool to update markets
	go r.OnChainRefreshRoutine(ctx)
	go r.CleanMarketsRoutine(ctx)
	// Loging Ethcalls per min
	go r.LogEthCallsPerMin(ctx)
	go r.LogState(ctx)
	go r.EventLoop(ctx)
	// 👇 bloque proprement
	<-ctx.Done()
}

func (r *Runner) ApiCallRoutine(ctx context.Context) {
	r.Cache.ApiCall(r.Conn.ClientHTTP)
}

func (r *Runner) WatchPositionRoutine(ctx context.Context) {
	r.Conn.WatchPositions(ctx)
}

func (r *Runner) OnChainRefreshRoutine(ctx context.Context) {
	utils.RunTicker(ctx, 5*time.Second, func() {
		OnChainRefresh(r.Conn, r.Cache)
	})
}

func (r *Runner) CleanMarketsRoutine(ctx context.Context) {
	utils.RunTicker(ctx, time.Minute, func() {
		_ = state.Filter(r.Cache.Markets, utils.WAD1DOT1)
	})
}

func (r *Runner) LogState(ctx context.Context) {
	utils.RunTicker(ctx, 10*time.Second, func() {
		logs := state.MarketReport(r.Cache.Markets, r.Cache.marketMap)
		r.Logger <- logs
	})
}

func (r *Runner) LogEthCallsPerMin(ctx context.Context) {
	r.Conn.LogsEthCallsFromLastMin(ctx, r.Logger)
}

func GetCandidates() []*engine.Liquidable {
	return nil
}

func (r *Runner) SimulateCandidatesRoutine(ctx context.Context) {
	candidates := GetCandidates()
	simulated := r.Engine.SimulateCandidates(r.Conn, r.Cache, r.Cache.marketMap, candidates, r.Logger)
	for _, l := range simulated {

	}

}

func (r *Runner) EventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-r.Conn.PositionCh:
			if !ok {
				return
			}
			r.Cache.ProcessEvents(event)
		}
	}
}

/*
func (r *Runner) FireLiquidationRoutine(ctx context.Context, conn *connector.Connector, errCh chan error, logChannel chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case liquidable := <-r.Cache.liquidCh:
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
