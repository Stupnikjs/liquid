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
	BaseLiquidatorAddr         = common.HexToAddress("0xAA5356884FE5aFA3DC7f2AA90e9C8E434fcCD87C")
	BaseLiquidatorUni          = common.HexToAddress("0xFa99159fC88E856738Ef3c02D09acDdfD99A3B33") // new one
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
	ArbitrumMorphoBlueAddr      = common.HexToAddress("0x6c247b1F6182318877311737BaC0844bAa518F5e")
	ArbitrumUniswapQuoterV2Addr = common.HexToAddress("0x61fFE014bA17989E743c5F6cB21bF9697530B21e")
	ArbitrumLiquidatorAddr      = common.HexToAddress("")
	ArbitrumUniswapV3Router     = common.HexToAddress("0xe592427a0aece92de3edee1f18e0157c05861564")
	ArbitrumWalletAddress       = common.HexToAddress("")

	// Optimism Addresses
	OptimismMorphoBlueAddr      = common.HexToAddress("0xce95AfbB8EA029495c66020883F87aaE8864AF92")
	OptimismLiquidatorAddr      = common.HexToAddress("")
	OptimismUniswapV3Router     = common.HexToAddress("0xE592427A0AEce92De3Edee1F18E0157C05861564")
	OptimismWalletAddress       = common.HexToAddress("")
	OptimismUniswapQuoterV2Addr = common.HexToAddress("0x61fFE014bA17989E743c5F6cB21bF9697530B21e")

	// Unichain
	UnichainWalletAddress       = common.HexToAddress("")
	UnichainUniswapV3Router     = common.HexToAddress("0x73855d06de49d0fe4a9c42636ba96c62da12ff9c")
	UnichainMorphoBlueAddr      = common.HexToAddress("0x8f5ae9CddB9f68de460C77730b018Ae7E04a140A")
	UnichainUniswapQuoterV2Addr = common.HexToAddress("0x385a5cf5f83e99f7bb2852b6a19c3538b9fa7658")
	UnichainLiquidatorAddr      = common.HexToAddress("")

	// WorldChain
	WorldChainWalletAddress       = common.HexToAddress("")
	WorldChainUniswapV3Router     = common.HexToAddress("0x091AD9e2e6e5eD44c1c66dB50e49A601F9f36cF6")
	WorldChainMorphoBlueAddr      = common.HexToAddress("0xE741BC7c34758b4caE05062794E8Ae24978AF432")
	WorldChainUniswapQuoterV2Addr = common.HexToAddress("0x10158D43e6cc414deE1Bd1eB0EfC6a5cBCfF244c")
	WorldChainLiquidatorAddr      = common.HexToAddress("")

	// Katana
	KatanaWalletAddress       = common.HexToAddress("")
	KatanaUniswapV3Router     = common.HexToAddress("0x4e1d81A3E627b9294532e990109e4c21d217376C")
	KatanaMorphoBlueAddr      = common.HexToAddress("0xD50F2DffFd62f94Ee4AEd9ca05C61d0753268aBc")
	KatanaUniswapQuoterV2Addr = common.HexToAddress("0x92dea23ED1C683940fF1a2f8fE23FE98C5d3041c")
	KatanaLiquidatorAddr      = common.HexToAddress("")
)
