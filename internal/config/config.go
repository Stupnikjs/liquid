package config

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/Stupnikjs/liquid/pkg/swap"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/joho/godotenv"
)

type Config struct {
	MorphoABI *morpho.MorphoABI
	Signer    *Signer
	Addresses Addresses
	ChainID   uint32
	Endpoints connector.RPCEndpoints
	Dexs      []swap.Dex
}

type Addresses struct {
	LiquidatorContract common.Address
	Wallet             common.Address
	Morpho             common.Address
}
type Signer struct {
	key    *ecdsa.PrivateKey
	signer types.Signer
}

func LoadBaseConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := 8453

	signer, err := NewBaseSigner(int64(chainid))
	if err != nil {
		fmt.Println(err)
	}
	morphoABI, err := morpho.LoadMorphoABI()

	return Config{
		MorphoABI: morphoABI,
		Signer:    signer,
		Addresses: Addresses{
			Wallet:             BaseWalletAddr,
			Morpho:             MorphoMain,
			LiquidatorContract: BaseLiquidatorNew,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("BASE_HTTP_RPC_ALCH"),
			Second:  os.Getenv("BASE_HTTP_RPC_DRPC"),
			WS:      os.Getenv("BASE_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(BaseUniswapQuoterV2Addr, BaseUniswapV3Router, 200*time.Millisecond, "UNIV3"),
			swap.NewUniDex(BasePankakeSwapQuoterV2Addr, BasePankakeSwapV3Router, 200*time.Millisecond, "PANCAKE"),
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
	morphoABI, err := morpho.LoadMorphoABI()

	return Config{
		MorphoABI: morphoABI,
		Signer:    signer,
		Addresses: Addresses{
			Wallet:             ArbitrumWalletAddress,
			Morpho:             ArbitrumMorphoBlueAddr,
			LiquidatorContract: ArbitrumLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("ARB_HTTP_RPC_ALCH"),
			Second:  os.Getenv("ARB_HTTP_RPC_DRPC"),
			WS:      os.Getenv("ARB_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(ArbitrumUniswapQuoterV2Addr, ArbitrumUniswapV3Router, 200*time.Millisecond, "UNIV3"),
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

	if err != nil {
		fmt.Println(err)
	}
	morphoABI, err := morpho.LoadMorphoABI()

	return Config{
		MorphoABI: morphoABI,
		Signer:    signer,
		Addresses: Addresses{
			Wallet:             KatanaWalletAddress,
			Morpho:             KatanaMorphoBlueAddr,
			LiquidatorContract: KatanaLiquidatorAddr,
		},
		ChainID: uint32(chainid),
		Endpoints: connector.RPCEndpoints{
			Primary: os.Getenv("KATANA_HTTP_RPC_ALCH"),
			Second:  os.Getenv("KATANA_HTTP_RPC_DRPC"),
			WS:      os.Getenv("KATANA_WS_RPC_ALCH"),
		},
		Dexs: []swap.Dex{
			swap.NewUniDex(KatanaUniswapQuoterV2Addr, KatanaUniswapV3Router, 200*time.Millisecond, "UNIV3"),
		},
	}
}
