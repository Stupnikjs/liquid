package config

import "github.com/ethereum/go-ethereum/common"

type Config struct {
	Signer             *Signer
	LiquidatorContract struct {
		Odos   common.Address
		Direct common.Address
	}
	Morpho        common.Address
	ChainID       int
	WalletAddress common.Address
	RPC           struct {
		HTTP []string
		WS   []string
	}
}

func LoadBaseConfig() Config {
	signer, _ := NewBaseSigner()
	return Config{
		Signer: signer,
		LiquidatorContract: struct {
			Odos   common.Address
			Direct common.Address
		}{
			Odos:   OdosRouterAddr,
			Direct: BaseUniswapV3Router,
		},
		Morpho:        BaseMorphoBlueAddr,
		ChainID:       8453,
		WalletAddress: BaseWalletAddr,
		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: BASE_HTTP_RPC,
			WS:   BASE_WS_RPC,
		},
	}
}
