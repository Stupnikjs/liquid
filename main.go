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
Deploy liquidator base avec Odos
Commencer le multichain Odos
Repenser les configs
*/

func main() {

	RunBase()
}

func RunBase() {

	conn := connector.NewConnector(config.BASE_HTTP_RPC, config.BASE_WS_RPC)
	// market from less than 10mounth
	markets := api.FilterMarket(conn.ClientHTTP)

	params := []morpho.MarketParams{}
	for _, m := range markets {
		params = append(params, m.MarketParams)
	}

	BaseSigner, err := config.NewSigner()
	if err != nil {
		fmt.Println(err)
	}

	cache := runner.NewCache(params)
	runner := runner.NewRunner(conn, cache, BaseSigner)
	runner.Init(context.Background())
	runner.Run(context.Background())
}

/*
func RunMain() {
	cexFeed := cex.NewCoinbaseConnector()

	conn := connector.NewConnector(config.MAIN_HTTP_RPC, config.MAIN_WS_RPC)
	MainSigner, err := morpho.NewSigner()
	if err != nil {
		fmt.Println(err)
	}
	baseConfig := morpho.ChainConfig{
		WalletAddress:        config.MainWalletAddr,
		LiquidatorAddress:    config.MainLiquidatorAddr,
		UniswapRouterAddress: config.MainUniswapV3Router,
		Signer:               MainSigner,
		Name:                 "main",
	}

	CacheConfig := core.NewCacheConfig(morpho.BaseParams, baseConfig)

	// base cache instance
	cache := core.NewCache(CacheConfig)

	cache.Scan(conn, cexFeed)
}
*/
