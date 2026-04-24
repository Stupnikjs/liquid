package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/cache"
	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/logging"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/swap"
)

type Runner struct {
	Cache       *cache.Cache
	Conn        *connector.Connector
	Logger      chan string
	Config      config.Config
	LiquidateCh chan cache.BorrowPosition
	// Config avec signer
}

func NewRunner(initedCache *cache.Cache, conf config.Config, logfile string) *Runner {
	conn := connector.NewConnector(conf.RPC.HTTP, conf.RPC.WS)
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
	r.OnChainRefreshAll()

	r.Cache.Markets.Range(func(id [32]byte) {
		snap := r.Cache.Markets.GetSnapshot(id)
		if snap == nil {
			return
		}
		morphoM := r.Cache.MarketMap[id]
		result, err := swap.QuoteBinarySearch(r.Conn.ClientHTTP, morphoM, r.Config.Addresses.UniSwapQuoter, snap.Stats.MaxCollateralPos, snap.Oracle.Price)
		if err != nil {
			r.Cache.Markets.Update(id, func(m *cache.Market) {
				m.Canceled = true
			})
			return
		}
		r.Cache.Markets.Update(id, func(m *cache.Market) {
			m.Stats.MaxUniSwappable = result.AmountIn
			m.Stats.SwapFee = result.Fee
			m.RecomputeHFUnsafe(len(m.Positions))
		})

	})

	var snap *cache.MarketSnapshot
	for k, v := range r.Cache.MarketMap {
		snap = r.Cache.Markets.GetSnapshot(k)
		if snap != nil {
			msg := fmt.Sprintf("_________ Market %s/%s in cache with %d pos ___________",
				v.CollateralTokenStr, v.LoanTokenStr, len(snap.Positions))
			fmt.Println(msg) // log direct pour confirmer que les marchés existent
			r.Logger <- msg
		} else {
			fmt.Println("snap nil for market", v.CollateralTokenStr) // voir si snap est nil
		}
	}

}

func (r *Runner) OnChainRefreshAll() {
	var wg sync.WaitGroup
	for _, id := range r.Cache.Markets.Ids() {
		wg.Add(1)
		go func(id [32]byte) {
			defer wg.Done()
			onchain.OnChainRefresh(r.Conn, r.Cache.Markets, r.Cache.GetMorphoMarketFromId(id), id, r.Config.Addresses.Morpho)
		}(id)
	}
	wg.Wait()
}
