package config

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

var (
	MorphoMain = common.HexToAddress("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb")

	// Base Addresses
	BaseUniswapQuoterV2Addr    = common.HexToAddress("0x3d4e44Eb1374240CE5F1B871ab261CD16335B76a")
	BaseUniswapV3Router        = common.HexToAddress("0x2626664c2603336E57B271c5C0b26F421741e481")
	BaseMorphoBlueAddr         = w3.A("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb") // Morpho Blue mainnet
	BaseLiquidatorAddr         = common.HexToAddress("0xAA5356884FE5aFA3DC7f2AA90e9C8E434fcCD87C")
	BaseLiquidatorAddrV2       = common.HexToAddress("0x2661C239C38AaB0d333Be91F999F7E69dD706504")
	BaseWalletAddr             = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")
	BaseLiquidatorOdosContract = common.HexToAddress("")
	IRM                        = common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687")
	OdosRouterAddr             = common.HexToAddress("0x19cEeAd7105607Cd444F5ad10dd51356436095a1")
	OdosRouterAddrV3           = common.HexToAddress("0x0D05a7D3448512B78fa8A9e46c4872C88C4a0D05")
	// Mainnet Addresses
	MainLiquidatorAddr     = common.HexToAddress("")
	MainMorphoBlueAddr     = w3.A("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb")
	MainWalletAddr         = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")
	MainLiquidatorOdosAddr = common.HexToAddress("")
	MainUniswapV3Router    = common.HexToAddress("0xE592427A0AEce92De3Edee1F18E0157C05861564")

	// Arbitrum Addresses
	ArbitrumUniswapQuoterV2Addr = common.HexToAddress("0x61fFE014bA17989E743c5F6cB21bF9697530B21e")
	ArbitrumLiquidatorAddr      = common.HexToAddress("")
	ArbitrumUniswapV3Router     = common.HexToAddress("0xe592427a0aece92de3edee1f18e0157c05861564")
	ArbitrumWalletAddress       = common.HexToAddress("")

	// Optimism Addresses
	OptimismLiquidatorAddr  = common.HexToAddress("")
	OptimismUniswapV3Router = common.HexToAddress("")
	OptimismWalletAddress   = common.HexToAddress("")
)
