package runner

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/logging"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/swap"
)

type Runner struct {
	Cache       *market.Cache
	Conn        *connector.Connector
	Logger      chan string
	Config      config.Config
	LiquidateCh chan market.BorrowPosition
	// Config avec signer
}

func NewRunner(cache *market.Cache, conf config.Config) *Runner {
	var logfile string
	if conf.ChainID == 8453 {
		logfile = "base.log"
	} else {
		logfile = "main.log"
	}
	conn := connector.NewConnector(conf.RPC.HTTP, conf.RPC.WS)
	logger := logging.NewLogger(context.Background(), logfile)
	return &Runner{
		Cache:       cache,
		Conn:        conn,
		Logger:      logger,
		Config:      conf,
		LiquidateCh: make(chan market.BorrowPosition, 1),
	}
}

func formatAmount(amount *big.Int, decimals uint8) string {
	if amount == nil {
		return "0"
	}
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	amountF := new(big.Float).SetInt(amount)
	result := new(big.Float).Quo(amountF, divisor)
	return result.Text('f', 4)
}

func (r *Runner) Init(ctx context.Context) {
	err := r.ApiCallRoutine(ctx)
	if err != nil {
		fmt.Println(err)
	}
	r.OnChainRefreshAll()
	// CHECK Liquidity on markets with uniswap
	r.Cache.Markets.Range(func(id [32]byte) {
		snap := r.Cache.Markets.GetSnapshot(id)
		if snap == nil {
			return
		}
		morphoM := r.Cache.MarketMap[id]
		result, err := swap.Quote(r.Conn.ClientHTTP, morphoM, snap.Stats.MaxCollateralPos, snap.Oracle.Price)
		if err != nil {
			r.Cache.Markets.Update(id, func(m *market.Market) {
				m.Canceled = true
			})
			return
		}
		fmt.Printf("Pair %s/%s | source: uniswap | slippage: %f%% | amountIn: %s %s\n",
			morphoM.CollateralTokenStr,
			morphoM.LoanTokenStr,
			result.Slippage,
			formatAmount(result.AmountIn, uint8(morphoM.CollateralTokenDecimals)),
			morphoM.CollateralTokenStr,
		)
		r.Cache.Markets.Update(id, func(m *market.Market) {
			m.Stats.MaxUniSwappable = result.AmountIn
			m.Stats.SwapFee = result.Fee
		})

	})
	r.Cache.Markets.Range(func(id [32]byte) {
		err := r.Cache.Markets.CleanNonSwap(id)
		if err != nil {
			r.Logger <- fmt.Sprintf("Error cleaning non-swap positions for market %s: %v", id, err)
		}
	})
	fmt.Println(len(r.Cache.Markets.Ids()))
}

func (r *Runner) OnChainRefreshAll() {
	var wg sync.WaitGroup
	for _, id := range r.Cache.Markets.Ids() {
		wg.Add(1)
		go func(id [32]byte) {
			defer wg.Done()
			onchain.OnChainRefresh(r.Conn, r.Cache.Markets, r.Cache.GetMorphoMarketFromId(id), id)
		}(id)
	}
	wg.Wait()
}
