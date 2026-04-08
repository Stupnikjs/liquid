package main

import (
	"fmt"

	"github.com/Stupnikjs/morpho-sepolia/internal/config"
	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/core"
	"github.com/Stupnikjs/morpho-sepolia/pkg/cex"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

func main() {
	RunBase()
}

func RunBase() {
	cexFeed := cex.NewCoinbaseConnector()
	/*
		Base Init
	*/
	conn := connector.NewConnector(config.BASE_HTTP_RPC, config.BASE_WS_RPC)
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

	CacheConfig := core.NewCacheConfig(morpho.BaseParams, baseConfig)

	// base cache instance
	cache := core.NewCache(CacheConfig)
	// cache.LogHotMarket()
	cache.Init(conn)
	cache.Scan(conn, cexFeed)
}

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
