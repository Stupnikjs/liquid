package morpho

import (
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/ethereum/go-ethereum/common"
)

var (
	BaseParams = []MarketParams{
		// 1. cbBTC / USDC — 86% LLTV (~$981M borrow)
		{
			ID:                      [32]byte(common.HexToHash("0x9103c3b4e834476c9a62ea009ba2c884ee42e94e6e314a26f04d312434191836")),
			ChainID:                 8453,
			LoanToken:               common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"),
			CollateralTokenStr:      "cbBTC",
			Oracle:                  common.HexToAddress("0x663BECd10daE6C4A3Dcd89F1d76c1174199639B9"),
			LLTV:                    utils.ParseBigInt("860000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 8,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 true,
		},
		// 2. WETH / USDC — 86% LLTV (~$51M borrow)
		{
			ID:                      [32]byte(common.HexToHash("0x8793cf302b8ffd655ab97bd1c695dbd967807e8367a65cb2f4edaf1380ba1bda")),
			ChainID:                 8453,
			LoanToken:               common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0x4200000000000000000000000000000000000006"),
			CollateralTokenStr:      "WETH",
			Oracle:                  common.HexToAddress("0xFEa2D58cEfCb9fcb597723c6bAE66fFE4193aFE4"),
			LLTV:                    utils.ParseBigInt("860000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 true,
		},
		// 3. wstETH / WETH — 94.5% LLTV (~$14M borrow)
		{
			ID:                      [32]byte(common.HexToHash("0x3a4048c64ba1b375330d376b1ce40e4047d03b47ab4d48af484edec9fec801ba")),
			ChainID:                 8453,
			LoanToken:               common.HexToAddress("0x4200000000000000000000000000000000000006"),
			LoanTokenStr:            "WETH",
			CollateralToken:         common.HexToAddress("0xc1CBa3fCea344f92D9239c08C0568f6F2F0ee452"),
			CollateralTokenStr:      "wstETH",
			Oracle:                  common.HexToAddress("0x4A11590e5326138B514E08A9B52202D42077Ca65"),
			LLTV:                    utils.ParseBigInt("945000000000000000"),
			LoanTokenDecimals:       18,
			CollateralTokenDecimals: 18,
			PoolFee:                 100,
			Correlated:              true,
			CexOnly:                 false,
		},
		// 4. cbXRP / USDC — 62.5% LLTV (~$11M borrow)
		{
			ID:                      [32]byte(common.HexToHash("0xd4a903dc6d949519060c7707f9604fdc9772c046e05c2e3a8fce0bd7196e4109")),
			ChainID:                 8453,
			LoanToken:               common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0xcb585250f852C6c6bf90434AB21A00f02833a4af"),
			CollateralTokenStr:      "cbXRP",
			Oracle:                  common.HexToAddress("0x031b2EFC8d70042Ac8d9f5c793c4149eC4b60fdE"),
			LLTV:                    utils.ParseBigInt("625000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 6,
			PoolFee:                 3000,
			Correlated:              false,
			CexOnly:                 true,
		},
		// 6. weETH / WETH — 94.5% LLTV (~$3M borrow)
		{
			ID:                      [32]byte(common.HexToHash("0xfd0895ba253889c243bf59bc4b96fd1e06d68631241383947b04d1c293a0cfea")),
			ChainID:                 8453,
			LoanToken:               common.HexToAddress("0x4200000000000000000000000000000000000006"),
			LoanTokenStr:            "WETH",
			CollateralToken:         common.HexToAddress("0x04C0599Ae5A44757c0af6F9eC3b93da8976c150A"),
			CollateralTokenStr:      "weETH",
			Oracle:                  common.HexToAddress("0xcE629400c6AEdb64f087CAC40Ae6a382AEEef490"),
			LLTV:                    utils.ParseBigInt("945000000000000000"),
			LoanTokenDecimals:       18,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              true,
			CexOnly:                 false,
		},

		{
			ID:                      [32]byte(common.HexToHash("0x1c21c59df9db44bf6f645d854ee710a8ca17b479451447e9f56758aee10a2fad")),
			ChainID:                 8453,
			LoanToken:               common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22"),
			CollateralTokenStr:      "cbETH",
			Oracle:                  common.HexToAddress("0xb40d93F44411D8C09aD17d7F88195eF9b05cCD96"),
			LLTV:                    utils.ParseBigInt("860000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 true,
		},
		{
			ID:                      [32]byte(common.HexToHash("0x84662b4f95b85d6b082b68d32cf71bb565b3f22f216a65509cc2ede7dccdfe8c")),
			ChainID:                 8453,
			LoanToken:               common.HexToAddress("0x4200000000000000000000000000000000000006"),
			LoanTokenStr:            "WETH",
			CollateralToken:         common.HexToAddress("0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22"),
			CollateralTokenStr:      "cbETH",
			Oracle:                  common.HexToAddress("0xB03855Ad5AFD6B8db8091DD5551CAC4ed621d9E6"),
			LLTV:                    utils.ParseBigInt("945000000000000000"),
			LoanTokenDecimals:       18,
			CollateralTokenDecimals: 18,
			PoolFee:                 100,
			Correlated:              true,
			CexOnly:                 false,
		},
		// WETH / wrsETH
		{
			ID:                      [32]byte(common.HexToHash("0x214c2bf3c899c913efda9c4a49adff23f77bbc2dc525af7c05be7ec93f32d561")),
			ChainID:                 8453,
			LoanToken:               common.HexToAddress("0x4200000000000000000000000000000000000006"),
			LoanTokenStr:            "WETH",
			CollateralToken:         common.HexToAddress("0xEDfa23602D0EC14714057867A78d01e94176BEA0"),
			CollateralTokenStr:      "wrsETH",
			Oracle:                  common.HexToAddress("0x55E6DE626D8b937782F08C8D3d9e54e340a78D0e"),
			LLTV:                    utils.ParseBigInt("945000000000000000"),
			LoanTokenDecimals:       18,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              true,
			CexOnly:                 false,
		},
		{ // 0x46415998764C29aB2a25CbeA6254146D50D22687
			ID:                      [32]byte(common.HexToHash("0x7acb1a83ad1d818fc68baddbdb61e1d9039d62590c8cd92915054dea54eb512d")),
			ChainID:                 8453,
			LoanToken:               common.HexToAddress("0x55380fe7A1910dFf29A47B622057ab4139DA42C5"), // fxUSD on Base
			LoanTokenStr:            "FXUSD",
			CollateralToken:         common.HexToAddress("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"), // cbBTC on Base
			CollateralTokenStr:      "cbBTC",
			Oracle:                  common.HexToAddress("0x3cb1b1862d4581656f14c06dE1bC0973a11EeF34"), // Market specific Oracle
			LLTV:                    utils.ParseBigInt("860000000000000000"),                           // LLTV determined at creation
			LoanTokenDecimals:       18,                                                                // FXUSD uses 18 decimals
			CollateralTokenDecimals: 8,                                                                 // cbBTC uses 8 decimals
			PoolFee:                 0,
			Correlated:              false,
			CexOnly:                 false,
		},
		{ // 0x46415998764C29aB2a25CbeA6254146D50D22687

			ID:                      [32]byte(common.HexToHash("0x87c7d6527a86fb5acef2e610c05c4a89810f1abc1af36498a5912167b78f6321")),
			ChainID:                 8453,
			LoanToken:               common.HexToAddress("0x55380fe7A1910dFf29A47B622057ab4139DA42C5"), // fxUSD on Base
			LoanTokenStr:            "FXUSD",
			CollateralToken:         common.HexToAddress("0xacfE6019Ed1A7Dc6f7B508C02d1b04ec88cC21bf"), // cbBTC on Base
			CollateralTokenStr:      "VVV",
			Oracle:                  common.HexToAddress("0xa7813fF2bdd188EbD2109D550f96EF912d1188FA"), // Market specific Oracle
			LLTV:                    utils.ParseBigInt("625000000000000000"),                           // LLTV determined at creation
			LoanTokenDecimals:       18,                                                                // FXUSD uses 18 decimals
			CollateralTokenDecimals: 18,                                                                // cbBTC uses 8 decimals
			PoolFee:                 0,
			Correlated:              false,
			CexOnly:                 false,
		},
		{
			// Tu peux calculer cet ID avec ComputeMarketId ou le récupérer depuis tes logs
			ID:                      [32]byte(common.HexToHash("0xbe67415847d61a6638cbe300c6cca3a5c34f5c2b1cfd594dd4f8ec21969c6fe2")),
			ChainID:                 8453,                                                              // Base
			LoanToken:               common.HexToAddress("0x55380fe7A1910dFf29A47B622057ab4139DA42C5"), // fxUSD
			LoanTokenStr:            "FXUSD",
			CollateralToken:         common.HexToAddress("0xc1CBa3fCea344f92D9239c08C0568f6F2F0ee452"), // wstETH
			CollateralTokenStr:      "wstETH",
			Oracle:                  common.HexToAddress("0xDD626EB9E24bbF4f641bf3502EA1C3d115aD2Fd8"),
			LLTV:                    utils.ParseBigInt("860000000000000000"), // 86%
			LoanTokenDecimals:       18,
			CollateralTokenDecimals: 18,
			PoolFee:                 0,
			Correlated:              false,
			CexOnly:                 false,
		},
		{
			ID:                      [32]byte(common.HexToHash("0x2e2548390b8894d2ffaaababebc2b5f2501920e03a4c611b63b46e19bfc6b75d")), // L'ID unique du marché (MarketByUniqueKey) n'était pas dans ton JSON, à générer si besoin.
			ChainID:                 8453,                                                                                             // Supposé Base au vu de l'USDC (0x833...)
			LoanToken:               common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),                                // USDC
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0x93919784C523f39CACaa98Ee0a9d96c3F32b593e"), // uniBTC
			CollateralTokenStr:      "uniBTC",
			Oracle:                  common.HexToAddress("0x9932FAbEdf44F52081a98f0Cc254ED4B22fBE3a3"),
			LLTV:                    utils.ParseBigInt("770000000000000000"), // 77% LLTV
			LoanTokenDecimals:       6,                                       // USDC utilise 6 décimales
			CollateralTokenDecimals: 8,                                       // uniBTC utilise 8 décimales
			PoolFee:                 0,
			Correlated:              false,
			CexOnly:                 false,
		},
		{
			ID:                      [32]byte(common.HexToHash("0x")), // ID à compléter si nécessaire
			ChainID:                 8453,
			LoanToken:               common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"), // USDC
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0x4bcaf180df5b13c0441FE41A66e9638A2a410C6D"), // HERMES
			CollateralTokenStr:      "HERMES",
			Oracle:                  common.HexToAddress("0x535BeD796890e17445542d3365ddF444E014daa2"),
			LLTV:                    utils.ParseBigInt("980000000000000000"), // 98% LLTV
			LoanTokenDecimals:       6,                                       // USDC (6 decimals)
			CollateralTokenDecimals: 18,                                      // HERMES (18 decimals)
			PoolFee:                 0,
			Correlated:              false,
			CexOnly:                 false,
		},
	}
)
