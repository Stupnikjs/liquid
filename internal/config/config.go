package config

import (
	"fmt"
	"log"
	"os"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/Stupnikjs/liquid/pkg/swap"
	"github.com/joho/godotenv"
)

func LoadArbitrumConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 42161
	signer, err := lqtypes.NewArbitrumSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			Wallet:             ArbitrumWalletAddress,
			Morpho:             ArbitrumMorphoBlueAddr,
			LiquidatorContract: ArbitrumLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("ARB_HTTP_RPC_ALCH"),
			Second:  os.Getenv("ARB_HTTP_RPC_ALCH"),
			WS:      os.Getenv("ARB_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			{
				QuoterAddr: ArbitrumUniswapQuoterV2Addr,
				RouterAddr: ArbitrumUniswapV3Router,
				Quoter:     swap.UniQuoter,
				Name:       "UNIV3",
			},
		},
	}
}
