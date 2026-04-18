package main

import (
	"context"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/market"
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

	Wrapper(config.LoadBaseConfig(), baseFilter)
}

func Wrapper(conf config.Config, filters api.MarketFilters) {

	conn := connector.NewConnector(conf.RPC.HTTP, conf.RPC.WS)
	// market from less than 10mounth

	cache := market.NewCache(conn, conf, filters)
	runn := runner.NewRunner(cache, conf)
	runn.Init(context.Background())
	runn.Run(context.Background())

}
