package config

import (
	"math/big"

	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/ethereum/go-ethereum/common"
)

var (
	MorphoMain = common.HexToAddress("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb")

	MainnetUniswapQuoterV2Addr = common.HexToAddress("0x61fFE014bA17989E743c5F6cB21bF9697530B21e")
	MainnetUniswapV3Router     = common.HexToAddress("0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45")
	MainnetPankakeQuoterV2Addr = common.HexToAddress("0xB048Bbc1Ee6b733FFfCFb9e9CeF7375518e25997")
	MainnetPankakeV3Router     = common.HexToAddress("0x1b81D678ffb9C0263b24A97847620C99d213eB14")
	// Base Addresses
	BaseUniswapQuoterV2Addr = common.HexToAddress("0x3d4e44Eb1374240CE5F1B871ab261CD16335B76a")
	BaseUniswapV3Router     = common.HexToAddress("0x2626664c2603336E57B271c5C0b26F421741e481")

	BasePankakeSwapV3Router     = common.HexToAddress("0x1b81D678ffb9C0263b24A97847620C99d213eB14")
	BasePankakeSwapQuoterV2Addr = common.HexToAddress("0xB048Bbc1Ee6b733FFfCFb9e9CeF7375518e25997")

	BaseAerodromeRouterAddr    = common.HexToAddress("0xcF77a3Ba9A5CA399B7c97c74d54e5b1Beb874E43")
	BaseLiquidatorAddr         = common.HexToAddress("0xAA5356884FE5aFA3DC7f2AA90e9C8E434fcCD87C")
	BaseLiquidatorNew          = common.HexToAddress("0x8BB59aa1667E46f0587cBBA34557bb604c27b5f8") // mutli hop
	BaseLiquidatorAddrV2       = common.HexToAddress("0x2661C239C38AaB0d333Be91F999F7E69dD706504")
	BaseWalletAddr             = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")
	BaseLiquidatorOdosContract = common.HexToAddress("")

	IRM = common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687")

	// Arbitrum Addresses
	ArbitrumMorphoBlueAddr      = common.HexToAddress("0x6c247b1F6182318877311737BaC0844bAa518F5e")
	ArbitrumUniswapQuoterV2Addr = common.HexToAddress("0x61fFE014bA17989E743c5F6cB21bF9697530B21e")
	ArbitrumLiquidatorAddr      = common.HexToAddress("0xAA5356884FE5aFA3DC7f2AA90e9C8E434fcCD87C") // multihop
	ArbitrumUniswapV3Router     = common.HexToAddress("0xe592427a0aece92de3edee1f18e0157c05861564")
	ArbitrumWalletAddress       = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")

	// Optimism Addresses
	OptimismMorphoBlueAddr      = common.HexToAddress("0xce95AfbB8EA029495c66020883F87aaE8864AF92")
	OptimismLiquidatorAddr      = common.HexToAddress("")
	OptimismUniswapV3Router     = common.HexToAddress("0xE592427A0AEce92De3Edee1F18E0157C05861564")
	OptimismWalletAddress       = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")
	OptimismUniswapQuoterV2Addr = common.HexToAddress("0x61fFE014bA17989E743c5F6cB21bF9697530B21e")

	// Unichain
	UnichainWalletAddress       = common.HexToAddress("")
	UnichainUniswapV3Router     = common.HexToAddress("0x73855d06de49d0fe4a9c42636ba96c62da12ff9c")
	UnichainMorphoBlueAddr      = common.HexToAddress("0x8f5ae9CddB9f68de460C77730b018Ae7E04a140A")
	UnichainUniswapQuoterV2Addr = common.HexToAddress("0x385a5cf5f83e99f7bb2852b6a19c3538b9fa7658")
	UnichainLiquidatorAddr      = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")

	// WorldChain
	WorldChainWalletAddress       = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")
	WorldChainUniswapV3Router     = common.HexToAddress("0x091AD9e2e6e5eD44c1c66dB50e49A601F9f36cF6")
	WorldChainMorphoBlueAddr      = common.HexToAddress("0xE741BC7c34758b4caE05062794E8Ae24978AF432")
	WorldChainUniswapQuoterV2Addr = common.HexToAddress("0x10158D43e6cc414deE1Bd1eB0EfC6a5cBCfF244c")
	WorldChainLiquidatorAddr      = common.HexToAddress("0xAA5356884FE5aFA3DC7f2AA90e9C8E434fcCD87C")

	// Katana
	KatanaWalletAddress       = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")
	KatanaUniswapV3Router     = common.HexToAddress("0x4e1d81A3E627b9294532e990109e4c21d217376C")
	KatanaMorphoBlueAddr      = common.HexToAddress("0xD50F2DffFd62f94Ee4AEd9ca05C61d0753268aBc")
	KatanaUniswapQuoterV2Addr = common.HexToAddress("0x92dea23ED1C683940fF1a2f8fE23FE98C5d3041c")
	KatanaLiquidatorAddr      = common.HexToAddress("0x2661C239C38AaB0d333Be91F999F7E69dD706504") // multihop

	HypeWalletAddress       = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")
	HypeUniswapV3Router     = common.HexToAddress("0xe8571fd6629da6e488f7bbd83e729c20fa9b97b4")
	HypeUniswapQuoterV2Addr = common.HexToAddress("0x6Cdcd65e03c1CEc3730AeeCd45bc140D57A25C77")
	HypeMorphoBlueAddr      = common.HexToAddress("0x68e37dE8d93d3496ae143F2E900490f6280C57cD")
	HypeLiquidatorAddress   = common.HexToAddress("0xAA5356884FE5aFA3DC7f2AA90e9C8E434fcCD87C")

	PolygonWalletAddress       = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")
	PolygonMorphoBlueAddr      = common.HexToAddress("0x1bF0c2541F820E775182832f06c0B7Fc27A25f67")
	PolygonLiquidatorAddr      = common.HexToAddress("0xAA5356884FE5aFA3DC7f2AA90e9C8E434fcCD87C")
	PolygonUniswapV3Router     = common.HexToAddress("0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45")
	PolygonUniswapQuoterV2Addr = common.HexToAddress("0xb27308f9F90D607463bb33eA1BeBb41C27CE5AB6")

	MonadWalletAddress       = common.HexToAddress("0x78D3FEc647f35E5D413597D217C5E0D9605acE3E")
	MonadUniswapV3Router     = common.HexToAddress("0xfe31f71c1b106eac32f1a19239c9a9a72ddfb900")
	MonadUniswapQuoterV2Addr = common.HexToAddress("0x661e93cca42afacb172121ef892830ca3b70f08d")
	MonadPankakeQuoterV2Addr = common.HexToAddress("0xB048Bbc1Ee6b733FFfCFb9e9CeF7375518e25997")
	MonadPankakeV3Router     = common.HexToAddress("0x1b81D678ffb9C0263b24A97847620C99d213eB14")
	MonadMorphoBlueAddr      = common.HexToAddress("0xD5D960E8C380B724a48AC59E2DfF1b2CB4a1eAee")
	MonadLiquidatorAddr      = common.HexToAddress("0xAA5356884FE5aFA3DC7f2AA90e9C8E434fcCD87C")
)

type BridgeToken struct {
	Address   common.Address
	MaxAmount *big.Int
}

var BaseBridgeTokens = []BridgeToken{
	{
		Address:   common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
		MaxAmount: new(big.Int).Mul(big.NewInt(10_000), big.NewInt(1e6)), // 10k USDC
	},
	{
		Address:   common.HexToAddress("0x4200000000000000000000000000000000000006"),
		MaxAmount: new(big.Int).Mul(big.NewInt(5), utils.TenPowInt(18)), // 5 WETH
	},
	{
		Address:   common.HexToAddress("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"),
		MaxAmount: big.NewInt(10_000_000), // 0.1 cbBTC
	},
	{
		Address:   common.HexToAddress("0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22"),
		MaxAmount: new(big.Int).Mul(big.NewInt(5), utils.TenPowInt(18)), // 5 cbETH
	},
	{
		Address:   common.HexToAddress("0xc1CBa3fCea344f92D9239c08C0568f6F2F0ee452"),
		MaxAmount: new(big.Int).Mul(big.NewInt(5), utils.TenPowInt(18)), // 5 wstETH
	},
	{
		Address:   common.HexToAddress("0x04C0599Ae5A44757c0AF6F9eC3b93da8976c150a"),
		MaxAmount: new(big.Int).Mul(big.NewInt(5), utils.TenPowInt(18)), // 5 weETH
	},
	{
		Address:   common.HexToAddress("0x60a3E35Cc302bFA44Cb288Bc5a4F316Fdb1adb42"),
		MaxAmount: new(big.Int).Mul(big.NewInt(10_000), utils.TenPowInt(6)), // 10k EURC
	},
	{
		Address:   common.HexToAddress("0x5d3a1Ff2b6BAb83b63cd9AD0787074081a52ef34"),
		MaxAmount: new(big.Int).Mul(big.NewInt(10_000), utils.TenPowInt(18)), // 10k USDe
	},
}
