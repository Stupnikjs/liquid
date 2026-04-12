package config

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

var (
	MorphoMain           = common.HexToAddress("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb")
	BaseUniswapV3Router  = common.HexToAddress("0x2626664c2603336E57B271c5C0b26F421741e481")
	BaseMorphoBlueAddr   = w3.A("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb") // Morpho Blue mainnet
	BaseLiquidatorAddr   = common.HexToAddress("0xAA5356884FE5aFA3DC7f2AA90e9C8E434fcCD87C")
	BaseLiquidatorAddrV2 = common.HexToAddress("0x2661C239C38AaB0d333Be91F999F7E69dD706504")
	BaseWalletAddr       = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")
	IRM                  = common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687")
	MainLiquidatorAddr   = common.HexToAddress("")
	MainMorphoBlueAddr   = w3.A("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb")
	MainWalletAddr       = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")
	MainUniswapV3Router  = common.HexToAddress("0xE592427A0AEce92De3Edee1F18E0157C05861564")
)
