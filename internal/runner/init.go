package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/cache"
	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/logging"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
)

type Runner struct {
	Cache       *cache.Cache
	Conn        *connector.Connector
	Logger      chan string
	Config      config.Config
	LiquidateCh chan cache.BorrowPosition
	// Config avec signer
}

func NewRunner(initedCache *cache.Cache, conn *connector.Connector, conf config.Config, logfile string) *Runner {

	logger := logging.NewLogger(context.Background(), logfile)
	return &Runner{
		Cache:       initedCache,
		Conn:        conn,
		Logger:      logger,
		Config:      conf,
		LiquidateCh: make(chan cache.BorrowPosition, 1),
	}
}

func (r *Runner) Init(ctx context.Context) {
	err := r.ApiCallRoutine(ctx)
	if err != nil {
		fmt.Println(err)
	}
	r.OnChainRefreshAll(ctx)

	for _, id := range r.Cache.Markets.Ids() {
		r.Swap(id)
	}
}

func (r *Runner) OnChainRefreshAll(ctx context.Context) {
	var wg sync.WaitGroup
	for _, id := range r.Cache.Markets.Ids() {
		wg.Add(1)
		go func(id [32]byte) {
			defer wg.Done()
			onchain.OnChainRefresh(r.Conn, ctx, r.Cache.Markets, r.Cache.GetMorphoMarketFromId(id), id, r.Config.Addresses.Morpho)
		}(id)
	}

	wg.Wait()

}

func (r *Runner) LogMarkets() {
	for _, id := range r.Cache.Markets.Ids() {
		m := r.Cache.GetMorphoMarketFromId(id)
		snap := r.Cache.Markets.GetSnapshot(id)
		if snap == nil {
			continue
		}
		stats := snap.Stats
		r.Logger <- fmt.Sprintf("%s/%s shares:%s totcoll:%s", m.CollateralTokenStr, m.LoanTokenStr, utils.FormatWAD(stats.TotalBorrowShares), utils.FormatDecimals(stats.TotalBorrowAssets, int(m.CollateralTokenDecimals)))
	}
}
