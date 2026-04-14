package config

import "github.com/ethereum/go-ethereum/common"

type Addresses struct {
	LiquidatorOdosContract common.Address
	LiquidatorSwapContract common.Address
	Odos                   common.Address
	Wallet                 common.Address
	Morpho                 common.Address
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
	signer, _ := NewBaseSigner()
	return Config{
		Signer: signer,
		Addresses: Addresses{
			Odos:                   OdosRouterAddr,
			LiquidatorSwapContract: BaseUniswapV3Router,
			LiquidatorOdosContract: BaseLiquidatorOdosContract,
			Morpho:                 BaseMorphoBlueAddr,
			Wallet:                 BaseWalletAddr,
		},
		ChainID: 8453,

		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: BASE_HTTP_RPC,
			WS:   BASE_WS_RPC,
		},
	}
}

func LoadMainnetConfig() Config {
	signer, _ := NewMainnetSigner()
	return Config{
		Signer: signer,
		Addresses: Addresses{
			Wallet:                 MainWalletAddr,
			LiquidatorOdosContract: MainLiquidatorOdosAddr,
			Odos:                   OdosRouterAddrV3,
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
