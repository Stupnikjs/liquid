package main

import (
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/internal/runner"
	"github.com/Stupnikjs/liquid/pkg/api"
)

func main() {
	var filter api.MarketFilters

	filter = api.MarketFilters{
		MaxUsdMarket: 10_000_000_000,
		MinUsdMarket: 2_000,
	}

	chainid := os.Args[1]
	chainid_int, err := strconv.Atoi(chainid)
	if err != nil {
		log.Panicln("Must pass integer as first arg")
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			var r *runner.Runner
			switch chainid_int {
			case 8453:
				r = runner.Wrapper(config.LoadBaseConfig(), filter)
			case 747474:
				r = runner.Wrapper(config.LoadKatanaConfig(), filter)
			case 42161:
				// arb marche pas
				r = runner.Wrapper(config.LoadArbitrumConfig(), filter)
			}
			ParseCmd(r)
		}

	}()

	wg.Wait()

}
