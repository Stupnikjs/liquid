package main

import (
	"context"
	"fmt"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/scanner"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

func main() {
	RunBase()
}

func RunBase() {

	/*
		Base Init
	*/
	conn := connector.NewConnector(config.BASE_HTTP_RPC, config.BASE_WS_RPC)
	// market from less than 10mounth
	markets := api.LogHotMarket(conn.ClientHTTP, 10)

	params := []morpho.MarketParams{}
	for _, m := range markets {
		params = append(params, m.MarketParams)
	}
	BaseSigner, err := morpho.NewSigner()
	if err != nil {
		fmt.Println(err)
	}
	baseConfig := morpho.ChainConfig{
		WalletAddress:        config.BaseWalletAddr,
		LiquidatorAddress:    config.BaseLiquidatorAddr,
		UniswapRouterAddress: config.BaseUniswapV3Router,
		Signer:               BaseSigner,
		Name:                 "base",
	}
	CacheConfig := baseConfig
	cache := scanner.NewCache(params, CacheConfig)
	runner := scanner.NewRunner(conn, cache)
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
