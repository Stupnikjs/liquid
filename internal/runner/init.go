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
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/Stupnikjs/liquid/pkg/swap"
	"github.com/ethereum/go-ethereum/core/types"
)

type Runner struct {
	MarketConsumer    *MarketConsumer
	QuoteConsumer     *QuoteConsumer
	LiquidateConsumer *liquidate.Consumer
}

type MarketCache interface {

	// API
	ApiCall() error
	ApiResyncRoutine(ctx context.Context)

	// Lecture
	Ids() [][32]byte
	GetSnapshot(id [32]byte) *cache.MarketSnapshot
	MorphoMarketByID(id [32]byte) morpho.MarketParams

	// Debug
	LogMarkets()
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
	err := r.MarketConsumer.ApiCall()
	if err != nil {
		log.Printf("initial api call error: %v", err)
	}
	r.MarketConsumer.OnChainRefreshInit(ctx)
	r.QuoteConsumer.QuotePools()
	r.MarketConsumer.Cache.LogMarkets()

}

// on chain call for all active market
// slow mode
func (c *MarketConsumer) OnChainRefreshInit(ctx context.Context) {
	var wg sync.WaitGroup
	for _, id := range c.Cache.Markets.Ids() {
		time.Sleep(100 * time.Millisecond)
		wg.Add(1)
		go func(id [32]byte) {
			m := c.Cache.MarketMap[id]
			defer wg.Done()
			oracleMethod := *c.Config.MorphoABI.Oracle.Price
			marketMethod := *c.Config.MorphoABI.Blue.Market

			err := onchain.OnChainRefresh(c.Conn, c.Config.Addresses.Morpho, ctx, c.Cache, m, &oracleMethod, &marketMethod, false)
			if err != nil {
				fmt.Println(err)
			}

		}(id)
	}

	wg.Wait()

}
