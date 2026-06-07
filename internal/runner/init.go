package runner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/internal/db"
	"github.com/Stupnikjs/liquid/internal/liquidate"

	"github.com/Stupnikjs/liquid/internal/onchain"
	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/Stupnikjs/liquid/pkg/swap"
	"github.com/ethereum/go-ethereum/core/types"
)

type Runner struct {
	MarketConsumer    *MarketConsumer
	QuoteConsumer     *QuoteConsumer
	LiquidateConsumer *liquidate.Consumer
}

type MarketConsumer struct {
	Conn    connector.Connector
	Config  config.Config
	Cache   *cache.Cache
	Store   *db.Store
	EventCh <-chan *types.Log
}

type QuoteConsumer struct {
	Conn   connector.Connector
	Config config.Config
	Cache  *cache.Cache
	Routes *swap.RouteCache
}

func NewMarketConsumer(conn connector.Connector, config config.Config, cache *cache.Cache, routes *swap.RouteCache, store *db.Store) *MarketConsumer {
	return &MarketConsumer{
		Conn:    conn,
		Config:  config,
		Cache:   cache,
		Store:   store,
		EventCh: make(<-chan *types.Log),
	}
}

func NewQuoteConsumer(conn connector.Connector, config config.Config, cache *cache.Cache, routes *swap.RouteCache) *QuoteConsumer {
	return &QuoteConsumer{
		Conn:   conn,
		Config: config,
		Cache:  cache,
		Routes: routes,
	}
}

func NewRunner(conn connector.Connector, config config.Config, routeCache *swap.RouteCache, mCache *cache.Cache) *Runner {
	liquidateCh := make(chan cache.BorrowPosition, 1)
	database, err := db.OpenDb(fmt.Sprintf("%d.db", config.ChainID))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	return &Runner{
		LiquidateConsumer: liquidate.NewConsumer(conn, config, mCache, routeCache, liquidateCh),
		QuoteConsumer:     NewQuoteConsumer(conn, config, mCache, routeCache),
		MarketConsumer:    NewMarketConsumer(conn, config, mCache, routeCache, &db.Store{DB: database, EntryToFlush: []db.Entry{}}),
	}
}

func (r *Runner) Init(ctx context.Context) {
	err := r.ApiCall()
	if err != nil {
		log.Printf("initial api call error: %v", err)
	}
	r.MarketConsumer.OnChainRefreshAll(ctx)
	r.QuoteConsumer.QuotePools()
	log.Println("Quoting over ")
	r.MarketConsumer.LogMarkets()

}

func (c *MarketConsumer) LogMarkets() {
	ids := c.Cache.Markets.Ids()
	for _, id := range ids {
		m := c.Cache.MarketMap[id]
		snap := c.Cache.Markets.GetSnapshot(id)
		if snap == nil {
			continue
		}
		fmt.Printf("[%s/%s] chain %d \n",
			m.CollateralTokenStr,
			m.LoanTokenStr,
			c.Config.ChainID)
	}
}

// on chain call for all active market
func (c *MarketConsumer) OnChainRefreshAll(ctx context.Context) {
	var wg sync.WaitGroup
	for _, id := range c.Cache.Markets.Ids() {
		time.Sleep(200 * time.Millisecond)
		wg.Add(1)
		go func(id [32]byte) {
			m := c.Cache.MarketMap[id]
			defer wg.Done()
			err := onchain.OnChainRefresh(c.Conn, c.Config.Addresses.Morpho, ctx, c.Cache, m, c.Config.MorphoABI.Oracle.Price, c.Config.MorphoABI.Blue.Market)
			if err != nil {
				fmt.Println(err)
			}

		}(id)
	}

	wg.Wait()

}
