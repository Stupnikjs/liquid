package config

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

var (
	MorphoMain      = common.HexToAddress("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb")
	UniswapV3Router = common.HexToAddress("0x2626664c2603336E57B271c5C0b26F421741e481")
	MorphoBlueAddr  = w3.A("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb") // Morpho Blue mainnet
	LiquidatorAddr  = common.HexToAddress("0xAA5356884FE5aFA3DC7f2AA90e9C8E434fcCD87C")
	WalletAddr      = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")
	IRM             = common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687")
)
