package main

import (
	"sync"
	"time"

	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/internal/runner"
	"github.com/Stupnikjs/liquid/pkg/api"
)

func main() {
	var baseFilter api.MarketFilters

	baseFilter = api.MarketFilters{
		MaxUsdMarket: 10_000_000_000,
		MinUsdMarket: 20_000,
	}

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		time.Sleep(200 * time.Second)
		defer wg.Done()
		// to avoid too much logs at the same time
		runner.Wrapper(config.LoadWorldChainConfig(), baseFilter, "world.log")
	}()

	go func() {
		defer wg.Done()
		// to avoid too much logs at the same time
		runner.Wrapper(config.LoadBaseConfig(), baseFilter, "base.log")
	}()

	go func() {
		time.Sleep(500 * time.Second)
		defer wg.Done()
		// to avoid too much logs at the same time
		runner.Wrapper(config.LoadArbitrumConfig(), baseFilter, "arb.log")
	}()

	wg.Wait()
}
