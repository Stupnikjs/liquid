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
	BaseSigner, err := morpho.NewSigner()
	if err != nil {
		fmt.Println(err)
	}
	baseConfig := morpho.ChainConfig{
		WalletAddress:        config.BaseWalletAddr,
		LiquidatorAddress:    config.BaseLiquidatorAddr,
		UniswapRouterAddress: config.BaseUniswapV3Router,
		Signer:               BaseSigner,
	}
	CacheConfig := core.NewCacheConfig(morpho.BaseParams, baseConfig)
	cexFeed := cex.NewCoinbaseConnector()
	cache := core.NewCache(CacheConfig)
	cache.Scan(conn, cexFeed)

}
