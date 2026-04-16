package main

import (
	"context"
	"fmt"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/runner"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

/*

 */

func main() {
	Wrapper(config.LoadBaseConfig())
}

func Wrapper(conf config.Config) {

	conn := connector.NewConnector(conf.RPC.HTTP, conf.RPC.WS)
	// market from less than 10mounth
	markets := api.FilterMarket(conn.ClientHTTP, conf.ChainID)
	fmt.Println("len market before init: ", len(markets))
	params := []morpho.MarketParams{}
	for _, m := range markets[:1] {
		params = append(params, m.MarketParams)
	}
	pos, _ := api.FetchBorrowersFromMarket(markets[0].ID, conf.ChainID)
	fmt.Println(len(pos))
	cache := runner.NewCache(params)
	runn := runner.NewRunner(cache, conf)
	runn.Init(context.Background())
	runn.Run(context.Background())

}
