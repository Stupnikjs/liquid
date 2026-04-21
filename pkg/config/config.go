package config

import (
	"log"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
)

type Addresses struct {
	LiquidatorContract common.Address
	SwapRouter         common.Address
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
	signer, _ := NewBaseSigner()
	return Config{
		Signer: signer,
		Addresses: Addresses{
			SwapRouter:         BaseUniswapV3Router,
			LiquidatorContract: BaseLiquidatorAddr,
			Morpho:             BaseMorphoBlueAddr,
			Wallet:             BaseWalletAddr,
		},
		ChainID: 8453,

		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: []string{os.Getenv("BASE_HTTP_RPC_ALCH")},
			WS:   []string{os.Getenv("BASE_WS_RPC_ALCH")},
		},
	}
}

func LoadMainnetConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	signer, _ := NewMainnetSigner()
	return Config{
		Signer: signer,
		Addresses: Addresses{
			Wallet:             MainWalletAddr,
			LiquidatorContract: MainLiquidatorOdosAddr,
			SwapRouter:         MainUniswapV3Router,
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
	signer, _ := NewArbitrumSigner()
	return Config{
		Signer: signer,
		Addresses: Addresses{
			Wallet:             ArbitrumWalletAddress,
			SwapRouter:         ArbitrumUniswapV3Router,
			LiquidatorContract: ArbitrumLiquidatorAddr,
		},
		ChainID: 42161,
		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: []string{os.Getenv("ARB_HTTP_RPC_ALCH"), os.Getenv("ARB_HTTP_RPC_ALCH")},
			WS:   []string{os.Getenv("ARB_WS_RPC_ALCH"), os.Getenv("ARB_WS_RPC_ALCH")},
		},
	}
}
