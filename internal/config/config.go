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

func LoadMainnetConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 1
	signer, err := lqtypes.NewBaseSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			Wallet:             BaseWalletAddr,
			Morpho:             MorphoMain,
			LiquidatorContract: BaseLiquidatorNew,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("MAINNET_HTTP_RPC_DRPC"),
			Second:  os.Getenv("MAINNET_HTTP_RPC_ALCH"),
			WS:      os.Getenv("MAINNET_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(MainnetUniswapQuoterV2Addr, MainnetUniswapV3Router, 300*time.Millisecond, "UNIV3"),
			swap.NewUniDex(MainnetPankakeQuoterV2Addr, MainnetPankakeV3Router, 300*time.Millisecond, "UNIV3"),
		},
	}
}

func LoadBaseConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 8453
	signer, err := lqtypes.NewBaseSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			Wallet:             BaseWalletAddr,
			Morpho:             MorphoMain,
			LiquidatorContract: BaseLiquidatorNew,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("BASE_HTTP_RPC_DRPC"),
			Second:  os.Getenv("BASE_HTTP_RPC_ALCH"),
			WS:      os.Getenv("BASE_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(BaseUniswapQuoterV2Addr, BaseUniswapV3Router, 200*time.Millisecond, "UNIV3"),
			swap.NewUniDex(BasePankakeSwapQuoterV2Addr, BasePankakeSwapV3Router, 200*time.Millisecond, "PANCAKE"),
		},
	}
}

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
			swap.NewUniDex(ArbitrumUniswapQuoterV2Addr, ArbitrumUniswapV3Router, 200*time.Millisecond, "UNIV3"),
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
			swap.NewUniDex(KatanaUniswapQuoterV2Addr, KatanaUniswapV3Router, 200*time.Millisecond, "UNIV3"),
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
			swap.NewUniDex(WorldChainUniswapQuoterV2Addr, WorldChainUniswapV3Router, 200*time.Millisecond, "UNIV3"),
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
			swap.NewUniDex(HypeUniswapQuoterV2Addr, HypeUniswapV3Router, 200*time.Millisecond, "UNIV3"),
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
			swap.NewUniDex(UnichainUniswapQuoterV2Addr, UnichainUniswapV3Router, 200*time.Millisecond, "UNIV3"),
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
			swap.NewUniDex(PolygonUniswapQuoterV2Addr, PolygonUniswapV3Router, 200*time.Millisecond, "UNIV3"),
		},
	}
}

func LoadOptimismConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 10
	signer, err := lqtypes.NewOptimismSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			Wallet:             OptimismWalletAddress,
			Morpho:             OptimismMorphoBlueAddr,
			LiquidatorContract: OptimismLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("OPT_HTTP_RPC_DRPC"),
			Second:  os.Getenv("OPT_HTTP_RPC_ALCH"),
			WS:      os.Getenv("OPT_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(OptimismUniswapQuoterV2Addr, OptimismUniswapV3Router, 200*time.Millisecond, "UNIV3"),
		},
	}
}

func LoadMonadConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 143
	signer, err := lqtypes.NewMonadSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			Wallet:             MonadWalletAddress,
			Morpho:             MonadMorphoBlueAddr,
			LiquidatorContract: MonadLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("MONAD_HTTP_RPC_DRPC"),
			Second:  os.Getenv("MONAD_HTTP_RPC_ALCH"),
			WS:      os.Getenv("MONAD_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(MonadUniswapQuoterV2Addr, MonadUniswapV3Router, 200*time.Millisecond, "UNIV3"),
			swap.NewUniDex(MonadUniswapQuoterV2Addr, MonadUniswapV3Router, 200*time.Millisecond, "PANKAKE"),
		},
	}
}
