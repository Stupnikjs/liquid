package main

import (
	"context"
	"sync"
	"time"

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
		MinUsdMarket: 20_000,
	}

	var wg sync.WaitGroup

	wg.Add(4)
	/*
		go func() {
			defer wg.Done()
			Wrapper(config.LoadKatanaConfig(), baseFilter, "katana.log")
		}()

		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Second) // to avoid too much logs at the same time
			Wrapper(config.LoadWorldChainConfig(), baseFilter, "world.log")
		}()
	*/
	go func() {
		defer wg.Done()
		time.Sleep(30 * time.Second) // to avoid too much logs at the same time
		Wrapper(config.LoadBaseConfig(), baseFilter, "base.log")
	}()
	/*
		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Second) // to avoid too much logs at the same time
			Wrapper(config.LoadUnichainConfig(), baseFilter, "uni.log")
		}()
	*/
	wg.Wait()
}

func Wrapper(conf config.Config, filters api.MarketFilters, logfile string) {

	conn := connector.NewConnector(conf.RPC.HTTP, conf.RPC.WS)
	// market from less than 10mounth

	cached := cache.NewCache(conn, conf, filters)
	runn := runner.NewRunner(cached, conn, conf, logfile)
	runn.Init(context.Background())
	runn.Run(context.Background())

}
