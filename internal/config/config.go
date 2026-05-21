package config

import (
	"fmt"
	"log"
	"os"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/internal/swap"
	"github.com/joho/godotenv"
)

func LoadBaseConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := int64(8453)
	signer, err := lqtypes.NewBaseSigner(chainid)
	if err != nil {
		fmt.Println(err)
	}

	dexs := []lqtypes.Dex{

		{
			QuoterAddr: BasePankakeSwapQuoterV2Addr,
			RouterAddr: BasePankakeSwapV3Router,
			Quoter:     swap.UniQuoter,
			Name:       "PANCAKE", // same abi than univ3
		},

		{
			QuoterAddr: BaseUniswapQuoterV2Addr,
			RouterAddr: BaseUniswapV3Router,
			Quoter:     swap.UniQuoter,
			Name:       "UNIV3",
		},

		/*
			{
				QuoterAddr: BaseAerodromeRouterAddr,
				RouterAddr: BaseAerodromeRouterAddr,
				Quoter:     swap.AerodromeQuoter,
				Name:       "AERO",
			},*/
	}
	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			UniSwapRouter:      BaseUniswapV3Router,
			UniSwapQuoter:      BaseUniswapQuoterV2Addr,
			LiquidatorContract: BaseLiquidatorUni,
			Morpho:             MorphoMain,
			Wallet:             BaseWalletAddr,
		},
		ChainID: uint32(chainid),

		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: []string{os.Getenv("BASE_HTTP_RPC_DRPC"), os.Getenv("BASE_HTTP_RPC_ALCH")},
			WS:   []string{os.Getenv("BASE_WS_RPC_DRPC"), os.Getenv("BASE_WS_RPC_ALCH")},
		},
		Dexs: dexs,
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
			UniSwapRouter:      ArbitrumUniswapV3Router,
			Morpho:             ArbitrumMorphoBlueAddr,
			UniSwapQuoter:      ArbitrumUniswapQuoterV2Addr,
			LiquidatorContract: ArbitrumLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: []string{os.Getenv("ARB_HTTP_RPC_ALCH"), os.Getenv("ARB_HTTP_RPC_ALCH")},
			WS:   []string{os.Getenv("ARB_WS_RPC_ALCH"), os.Getenv("ARB_WS_RPC_ALCH")},
		},
		Dexs: []lqtypes.Dex{
			{
				QuoterAddr: ArbitrumUniswapQuoterV2Addr,
				RouterAddr: ArbitrumUniswapV3Router,
				Quoter:     swap.UniQuoter,
				Name:       "UNIV3",
			},
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
		// Change here
		Addresses: lqtypes.Addresses{
			Wallet:             OptimismWalletAddress,
			UniSwapRouter:      OptimismUniswapV3Router,
			UniSwapQuoter:      OptimismUniswapQuoterV2Addr,
			LiquidatorContract: OptimismLiquidatorAddr,
			Morpho:             OptimismMorphoBlueAddr,
		},
		ChainID: uint32(chainid),
		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: []string{os.Getenv("OPT_HTTP_RPC_ALCH"), os.Getenv("OPT_HTTP_RPC_ALCH")},
			WS:   []string{os.Getenv("OPT_WS_RPC_ALCH"), os.Getenv("OPT_WS_RPC_ALCH")},
		},
		Dexs: []lqtypes.Dex{
			{
				QuoterAddr: OptimismUniswapQuoterV2Addr,
				RouterAddr: OptimismUniswapV3Router,
				Quoter:     swap.UniQuoter,
				Name:       "UNIV3",
			},
		},
	}
}

func LoadUnichainConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 130
	signer, err := lqtypes.NewUniChainSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			Wallet:             UnichainWalletAddress,
			UniSwapRouter:      UnichainUniswapV3Router,
			Morpho:             UnichainMorphoBlueAddr,
			UniSwapQuoter:      UnichainUniswapQuoterV2Addr,
			LiquidatorContract: UnichainLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: []string{os.Getenv("UNICHAIN_HTTP_RPC_ALCH"), os.Getenv("UNICHAIN_HTTP_RPC_ALCH")},
			WS:   []string{os.Getenv("UNICHAIN_WS_RPC_ALCH"), os.Getenv("UNICHAIN_WS_RPC_ALCH")},
		},
	}
}

func LoadWorldChainConfig() lqtypes.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 480
	signer, _ := lqtypes.NewWorldChainSigner(int64(chainid))
	return lqtypes.Config{
		Signer: signer,
		Addresses: lqtypes.Addresses{
			Wallet:             WorldChainWalletAddress,
			UniSwapRouter:      WorldChainUniswapV3Router,
			Morpho:             WorldChainMorphoBlueAddr,
			UniSwapQuoter:      WorldChainUniswapQuoterV2Addr,
			LiquidatorContract: WorldChainLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: []string{os.Getenv("WORLDCHAIN_HTTP_RPC_DRPC"), os.Getenv("WORLDCHAIN_HTTP_RPC_ALCH")},
			WS:   []string{os.Getenv("WORLDCHAIN_WS_RPC_ALCH"), os.Getenv("WORLDCHAIN_WS_RPC_ALCH")},
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
			UniSwapRouter:      KatanaUniswapV3Router,
			Morpho:             KatanaMorphoBlueAddr,
			UniSwapQuoter:      KatanaUniswapQuoterV2Addr,
			LiquidatorContract: KatanaLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: []string{os.Getenv("KATANA_HTTP_RPC_DRPC"), os.Getenv("KATANA_HTTP_RPC_ALCH")},
			WS:   []string{os.Getenv("KATANA_WS_RPC_ALCH"), os.Getenv("KATANA_WS_RPC_ALCH")},
		},
	}
}
