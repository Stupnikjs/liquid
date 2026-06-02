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
	/*
		go func() {
			time.Sleep(200 * time.Second)
			defer wg.Done()
			// to avoid too much logs at the same time
			runner.Wrapper(config.LoadWorldChainConfig(), baseFilter, "world.log")
		}()
	*/
	go func() {
		defer wg.Done()
		time.Sleep(600 * time.Second)
		// to avoid too much logs at the same time
		runner.Wrapper(config.LoadArbitrumConfig(), arbitrum, "arb.log")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(300 * time.Second)
		// to avoid too much logs at the same time
		runner.Wrapper(config.LoadKatanaConfig(), arbitrum, "katana.log")
	}()

	go func() {
		defer wg.Done()

		// to avoid too much logs at the same time
		runner.Wrapper(config.LoadHypeChainConfig(), arbitrum, "hype.log")
	}()

	wg.Wait()
}
