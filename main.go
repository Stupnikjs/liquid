package main

import (
	"sync"
	"time"

	"github.com/Stupnikjs/liquid/internal/runner"
	"github.com/Stupnikjs/liquid/pkg/api"
	"github.com/Stupnikjs/liquid/pkg/config"
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
		time.Sleep(400 * time.Second)
		defer wg.Done()
		runner.Wrapper(config.LoadKatanaConfig(), baseFilter, "katana.log")
	}()
	/*
		go func() {
			time.Sleep(50 * time.Second)
			defer wg.Done()
			Wrapper(config.LoadMonadConfig(), baseFilter, "monad.log")
		}()
	*/
	go func() {
		defer wg.Done()
		time.Sleep(200 * time.Second) // to avoid too much logs at the same time
		runner.Wrapper(config.LoadWorldChainConfig(), baseFilter, "world.log")
	}()

	go func() {
		defer wg.Done()
		runner.Wrapper(config.LoadBaseConfig(), baseFilter, "base.log")
	}()

	wg.Wait()
}
