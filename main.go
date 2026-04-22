package main

import (
	"context"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/cache"
	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/runner"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
)

/*
Snap shot s'initialize pas
*/

func main() {
	var baseFilter api.MarketFilters

	baseFilter = api.MarketFilters{
		MaxUsdMarket: 10_000_000_000,
		MinUsdMarket: 10_000,
	}

	var wg sync.WaitGroup

	wg.Add(2)
	/*
		go func() {
			defer wg.Done()
			Wrapper(config.LoadBaseConfig(), baseFilter, "base.log")
		}()

	*/
	go func() {
		defer wg.Done()
		Wrapper(config.LoadArbitrumConfig(), baseFilter, "arb.log")
	}()

	wg.Wait()
}

func Wrapper(conf config.Config, filters api.MarketFilters, logfile string) {

	conn := connector.NewConnector(conf.RPC.HTTP, conf.RPC.WS)
	// market from less than 10mounth

	cached := cache.NewCache(conn, conf, filters)
	runn := runner.NewRunner(cached, conf, logfile)
	runn.Init(context.Background())
	runn.Run(context.Background())

}
