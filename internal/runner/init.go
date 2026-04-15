package runner

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/engine"
	"github.com/Stupnikjs/morpho-sepolia/internal/logging"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/swap"
)

type Runner struct {
	Cache  *Cache
	Engine *engine.Engine
	Conn   *connector.Connector
	Logger chan string
	Config config.Config
	// Config avec signer
}

func NewRunner(cache *Cache, conf config.Config) *Runner {
	var logfile string
	if conf.ChainID == 8543 {
		logfile = "base.log"
	} else {
		logfile = "main.log"
	}
	conn := connector.NewConnector(conf.RPC.HTTP, conf.RPC.WS)
	logger := logging.NewLogger(context.Background(), logfile)
	return &Runner{
		Cache:  cache,
		Engine: engine.NewEngine(conn, conf, logger),
		Conn:   conn,
		Logger: logger,
	}
}

func (r *Runner) Init(ctx context.Context) {
	r.ApiCallRoutine(ctx)
	r.OnChainRefreshAll()
	r.FilterMarketBySlippage(ctx)
	fmt.Println("len market after init: ", len(r.Cache.Markets.Ids()))

}

/* Only into init func no concurencie */
func (r *Runner) FilterMarketBySlippage(ctx context.Context) {
	for _, id := range r.Cache.Markets.Ids() {

		snap := r.Cache.Markets.GetSnapshot(id)
		marketP := r.Cache.marketMap[id]

		if snap == nil || snap.Oracle.Price.Sign() == 0 || snap.Oracle.Price == nil {
			r.Cache.Markets.Update(id, func(m *market.Market) {
				m.Canceled = true
			}) // pas d'oracle → inutilisable
			continue
		}

		var testAmount *big.Int
		if strings.Contains(marketP.CollateralTokenStr, "ETH") || strings.Contains(marketP.CollateralTokenStr, "BTC") {
			testAmount = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(marketP.CollateralTokenDecimals)), nil)
			testAmount.Mul(testAmount, big.NewInt(1))

		} else {
			testAmount = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(marketP.CollateralTokenDecimals)), nil)
			testAmount = testAmount.Mul(testAmount, big.NewInt(1000))
			// montant test : 10k$ en unités du collateral
		}

		priceImpact, oracleSlipage := swap.FindBestPool(
			r.Conn.ClientHTTP,
			marketP,
			testAmount,
			snap.Oracle.Price,
		)
		fmt.Println(marketP.CollateralTokenStr, priceImpact)

		if priceImpact > 2.0 || oracleSlipage > 3.0 {
			r.Cache.Markets.Update(id, func(m *market.Market) {
				m.Canceled = true
			})
			continue
		}

	}
}

func (r *Runner) OnChainRefreshAll() {
	var wg sync.WaitGroup
	for _, id := range r.Cache.Markets.Ids() {
		wg.Add(1)
		go func(id [32]byte) {
			defer wg.Done()
			onchain.OnChainRefresh(r.Conn, r.Cache.Markets, r.Cache.marketMap[id], id)
		}(id)
	}
	wg.Wait() // Init blocks until all markets have data
}
