package main

import (
	"context"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/runner"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

/*
Deploy liquidator base avec Odos
Commencer le multichain Odos
Repenser les configs

*/

func main() {
	Wrapper(config.LoadMainnetConfig())
}

func Wrapper(conf config.Config) {

	conn := connector.NewConnector(conf.RPC.HTTP, conf.RPC.WS)
	// market from less than 10mounth
	markets := api.FilterMarket(conn.ClientHTTP, conf.ChainID)

	params := []morpho.MarketParams{}
	for _, m := range markets {
		params = append(params, m.MarketParams)
	}

	cache := runner.NewCache(params)
	runner := runner.NewRunner(cache, conf)
	runner.Init(context.Background())
	runner.Run(context.Background())
}
