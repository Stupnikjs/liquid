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

	conn := connector.NewConnector(config.BASE_HTTP_RPC, config.BASE_WS_RPC)
	signer, err := morpho.NewSigner()
	if err != nil {
		fmt.Println(err)
	}
	chainConfig := morpho.ChainConfig{
		WalletAddress:        config.WalletAddr,
		LiquidatorAddress:    config.LiquidatorAddr,
		UniswapRouterAddress: config.UniswapV3Router,
		Signer:               signer,
	}
	CacheConfig := core.NewCacheConfig(morpho.BaseParams, chainConfig)
	cexFeed := cex.NewCoinbaseConnector()
	cache := core.NewCache(CacheConfig)
	cache.Scan(conn, cexFeed)

}
