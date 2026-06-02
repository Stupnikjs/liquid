package runner

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/db"
	"github.com/Stupnikjs/liquid/internal/liquidate"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/internal/onchain"
	"github.com/Stupnikjs/liquid/pkg/swap"
	"github.com/ethereum/go-ethereum/core/types"
)

type Runner struct {
	LiquidateConsumer *liquidate.Consumer
	Cache             *cache.Cache
	Routes            *swap.RouteCache
	Infra             *lqtypes.Infra
	DB                *sql.DB
	EntryToFlush      []db.Entry
	LiquidateCh       chan cache.BorrowPosition
	EventCh           <-chan *types.Log
}

func NewRunner(infra *lqtypes.Infra, routeCache *swap.RouteCache, mCache *cache.Cache) *Runner {
	liquidateCh := make(chan cache.BorrowPosition, 1)
	database, err := db.OpenDb(fmt.Sprintf("%d.db", infra.Config.ChainID))
	if err != nil {
		log.Fatal(err)
	}
	return &Runner{
		LiquidateConsumer: liquidate.NewConsumer(infra, mCache, routeCache, liquidateCh),
		Cache:             mCache,
		Routes:            routeCache,
		Infra:             infra,
		DB:                database,
		LiquidateCh:       liquidateCh,
		EventCh:           make(<-chan *types.Log),
		EntryToFlush:      make([]db.Entry, 0, MAX_FLUSH_QUEUE_SIZE),
	}
}

func (r *Runner) Init(ctx context.Context) {
	err := r.ApiCall()
	if err != nil {
		log.Printf("initial api call error: %v", err)
	}
	r.OnChainRefreshAll(ctx)
	r.QuotePools()
	log.Println("Quoting over ")
	r.LogMarkets()

}

func (r *Runner) LogMarkets() {
	ids := r.Cache.Markets.Ids()
	for _, id := range ids {
		m := r.Cache.MarketMap[id]
		snap := r.Cache.Markets.GetSnapshot(id)
		if snap == nil {
			continue
		}
		fmt.Printf("[%s/%s] chain %d \n",
			m.CollateralTokenStr,
			m.LoanTokenStr,
			r.Infra.Config.ChainID)
	}
}

// on chain call for all active market
func (r *Runner) OnChainRefreshAll(ctx context.Context) {
	var wg sync.WaitGroup
	for _, id := range r.Cache.Markets.Ids() {
		wg.Add(1)
		go func(id [32]byte) {
			m := r.Cache.MarketMap[id]
			defer wg.Done()
			err := onchain.OnChainRefresh(r.Infra, ctx, r.Cache, m, id)
			if err != nil {
				fmt.Println(err)
			}

		}(id)
	}

	wg.Wait()

}
