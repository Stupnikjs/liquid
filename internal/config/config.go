package config

import (
	"fmt"
	"log"
	"os"
	"time"

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
			Primary: os.Getenv("ARB_HTTP_RPC_DRPC"),
			Second:  os.Getenv("ARB_HTTP_RPC_ALCH"),
			WS:      os.Getenv("ARB_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(ArbitrumUniswapQuoterV2Addr, ArbitrumUniswapV3Router, 200*time.Millisecond),
		},
	}
}

func LoadKatanaConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 747474
	signer, err := lqtypes.NewKatanaSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			Wallet:             KatanaWalletAddress,
			Morpho:             KatanaMorphoBlueAddr,
			LiquidatorContract: KatanaLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("KATANA_HTTP_RPC_DRPC"),
			Second:  os.Getenv("KATANA_HTTP_RPC_ALCH"),
			WS:      os.Getenv("KATANA_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(KatanaUniswapQuoterV2Addr, KatanaUniswapV3Router, 200*time.Millisecond),
		},
	}
}

func LoadWorldChainConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 747474
	signer, err := lqtypes.NewWorldChainSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			Wallet:             WorldChainWalletAddress,
			Morpho:             WorldChainMorphoBlueAddr,
			LiquidatorContract: WorldChainLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("WORLD_HTTP_RPC_DRPC"),
			Second:  os.Getenv("WORLD_HTTP_RPC_ALCH"),
			WS:      os.Getenv("WORLD_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(WorldChainUniswapQuoterV2Addr, WorldChainUniswapV3Router, 200*time.Millisecond),
		},
	}
}

func LoadHypeChainConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 999
	signer, err := lqtypes.NewHypeSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			Wallet:             HypeWalletAddress,
			Morpho:             HypeMorphoBlueAddr,
			LiquidatorContract: HypeLiquidatorAddress,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("HYPE_HTTP_RPC_DRPC"),
			Second:  os.Getenv("HYPE_HTTP_RPC_ALCH"),
			WS:      os.Getenv("HYPE_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(HypeUniswapQuoterV2Addr, HypeUniswapV3Router, 200*time.Millisecond),
		},
	}
}

func LoadUnichainConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 456
	signer, err := lqtypes.NewUnichainSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			Wallet:             UnichainWalletAddress,
			Morpho:             UnichainMorphoBlueAddr,
			LiquidatorContract: UnichainLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("UNICHAIN_HTTP_RPC_DRPC"),
			Second:  os.Getenv("UNICHAIN_HTTP_RPC_ALCH"),
			WS:      os.Getenv("UNICHAIN_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(UnichainUniswapQuoterV2Addr, UnichainUniswapV3Router, 200*time.Millisecond),
		},
	}
}

func LoadPolygonConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 137
	signer, err := lqtypes.NewPolygonSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			Wallet:             PolygonWalletAddress,
			Morpho:             PolygonMorphoBlueAddr,
			LiquidatorContract: PolygonLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("POLYGON_HTTP_RPC_DRPC"),
			Second:  os.Getenv("POLYGON_HTTP_RPC_ALCH"),
			WS:      os.Getenv("POLYGON_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(PolygonUniswapQuoterV2Addr, PolygonUniswapV3Router, 200*time.Millisecond),
		},
	}
}
