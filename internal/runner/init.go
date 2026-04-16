package runner

import (
	"context"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/logging"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
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
	if conf.ChainID == 8543 {
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
	r.ApiCallRoutine(ctx)
	r.OnChainRefreshAll()
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
	wg.Wait() // Init blocks until all markets have data
}
