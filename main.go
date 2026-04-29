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
			time.Sleep(200 * time.Second)
			defer wg.Done()
			Wrapper(config.LoadKatanaConfig(), baseFilter, "katana.log")
		}()
	*/
	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Second) // to avoid too much logs at the same time
		Wrapper(config.LoadWorldChainConfig(), baseFilter, "world.log")
	}()

	go func() {
		defer wg.Done()
		Wrapper(config.LoadBaseConfig(), baseFilter, "base.log")
	}()

	wg.Wait()
}

func Wrapper(conf config.Config, filters api.MarketFilters, logfile string) {

	conn := connector.NewConnector(conf.RPC.HTTP, conf.RPC.WS)

	cached := cache.NewCache(conn, conf, filters)
	runn := runner.NewRunner(cached, conn, conf, logfile)
	runn.Init(context.Background())
	runn.Run(context.Background())

}
