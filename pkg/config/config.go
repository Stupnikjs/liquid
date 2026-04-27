package config

import (
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
)

type Addresses struct {
	LiquidatorContract common.Address
	UniSwapRouter      common.Address
	UniSwapQuoter      common.Address
	Wallet             common.Address
	Morpho             common.Address
}

type Config struct {
	Signer    *Signer
	Addresses Addresses
	Morpho    common.Address
	ChainID   uint32
	RPC       struct {
		HTTP []string
		WS   []string
	}
}

func LoadBaseConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := int64(8453)
	signer, err := NewBaseSigner(common.HashLength)
	if err != nil {
		fmt.Println(err)
	}

	return Config{
		Signer: signer,
		Addresses: Addresses{
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
			HTTP: []string{os.Getenv("BASE_HTTP_RPC_ALCH"), os.Getenv("BASE_HTTP_RPC_ALCH")},
			WS:   []string{os.Getenv("BASE_WS_RPC_ALCH"), os.Getenv("BASE_WS_RPC_ALCH")},
		},
	}
}

func LoadMainnetConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	signer, err := NewMainnetSigner()
	if err != nil {
		fmt.Println(err)
	}

	return Config{
		Signer: signer,
		Addresses: Addresses{
			Wallet:             MainWalletAddr,
			LiquidatorContract: MainLiquidatorOdosAddr,
			UniSwapRouter:      MainUniswapV3Router,
			UniSwapQuoter:      MainUniswapV3Router, // change
		},
		ChainID: 1,
		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: MAIN_HTTP_RPC,
			WS:   MAIN_WS_RPC,
		},
	}
}

func LoadArbitrumConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 42161
	signer, err := NewArbitrumSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return Config{
		Signer: signer,
		Addresses: Addresses{
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
	}
}

func LoadOptimismConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 10
	signer, err := NewOptimismSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return Config{
		Signer: signer,
		// Change here
		Addresses: Addresses{
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
	}
}

func LoadUnichainConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 130
	signer, err := NewUniChainSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return Config{
		Signer: signer,
		Addresses: Addresses{
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

func LoadWorldChainConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 480
	signer, _ := NewWorldChainSigner(int64(chainid))
	return Config{
		Signer: signer,
		Addresses: Addresses{
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
			HTTP: []string{os.Getenv("WORLDCHAIN_HTTP_RPC_ALCH"), os.Getenv("WORLDCHAIN_HTTP_RPC_ALCH")},
			WS:   []string{os.Getenv("WORLDCHAIN_WS_RPC_ALCH"), os.Getenv("WORLDCHAIN_WS_RPC_ALCH")},
		},
	}
}

func LoadKatanaConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 747474
	signer, err := NewKatanaSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}

	return Config{
		Signer: signer,
		Addresses: Addresses{
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
			HTTP: []string{os.Getenv("KATANA_HTTP_RPC_ALCH"), os.Getenv("KATANA_HTTP_RPC_ALCH")},
			WS:   []string{os.Getenv("KATANA_WS_RPC_ALCH"), os.Getenv("KATANA_WS_RPC_ALCH")},
		},
	}
}
