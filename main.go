package main

import (
	"context"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/runner"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
)

/*

 */

func main() 
	Wrapper(config.LoadBaseConfig(), api.MarketFilters{})
}

func Wrapper(conf config.Config, filters api.MarketFilters) {

	conn := connector.NewConnector(conf.RPC.HTTP, conf.RPC.WS)
	// market from less than 10mounth

	cache := runner.NewCache(conn, conf, )
	runn := runner.NewRunner(cache, conf)
	runn.Init(context.Background())
	runn.Run(context.Background())

}
