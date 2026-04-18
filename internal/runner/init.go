package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/logging"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/swap"
)

type Runner struct {
	Cache  *market.Cache
	Conn   *connector.Connector
	Logger chan string
	Config config.Config
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
		Cache:  cache,
		Conn:   conn,
		Logger: logger,
		Config: conf,
	}
}

func (r *Runner) Init(ctx context.Context) {
	err := r.ApiCallRoutine(ctx)
	if err != nil {
		fmt.Println(err)
	}
	r.OnChainRefreshAll()
	r.Cache.Markets.Range(func(id [32]byte) {
		snap := r.Cache.Markets.GetSnapshot(id)
		morphoM := r.Cache.MarketMap[id]
		amountIn, result, bestSlippage := swap.FindBestPool(
			r.Conn.ClientHTTP, morphoM, snap.Stats.MaxCollateralPos, snap.Oracle.Price,
		)

		if amountIn == nil {
			fmt.Printf("Pair %s/%s — no liquid pool\n", morphoM.CollateralTokenStr, morphoM.LoanTokenStr)
			return
		}

		fmt.Printf("Pair %s/%s | source: %s | fee: %d | stable: %v | slip: %.6f%% | amountIn: %s\n",
			morphoM.CollateralTokenStr, morphoM.LoanTokenStr,
			result.Source, result.Fee, result.Stable,
			bestSlippage, amountIn.String())
	})
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
