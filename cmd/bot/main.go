package main

import (
	"sync"
	"time"

	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/internal/runner"
	"github.com/Stupnikjs/liquid/pkg/api"
)

func main() {
	var arbitrum api.MarketFilters

	arbitrum = api.MarketFilters{
		MaxUsdMarket: 10_000_000_000,
		MinUsdMarket: 20_000,
	}

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Second)
		// to avoid too much logs at the same time
		runner.Wrapper(config.LoadArbitrumConfig(), arbitrum)
	}()

	go func() {
		defer wg.Done()
		time.Sleep(200 * time.Second)
		// to avoid too much logs at the same time
		runner.Wrapper(config.LoadKatanaConfig(), arbitrum)
	}()

	wg.Wait()
}
