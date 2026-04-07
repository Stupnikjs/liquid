package morpho

import (
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/ethereum/go-ethereum/common"
)

var (
	MAINNET_MARKETS = []MarketParams{ // wstETH/USDT - $167M
		{
			ID:                      [32]byte(common.HexToHash("0xe7e9694b754c4d4f7e21faf7223f6fa71abaeb10296a4c43a54a7977149687d2")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
			LoanTokenStr:            "USDT",
			CollateralToken:         common.HexToAddress("0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0"),
			CollateralTokenStr:      "wstETH",
			Oracle:                  common.HexToAddress("0x95DB30fAb9A3754e42423000DF27732CB2396992"),
			LLTV:                    utils.ParseBigInt("860000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 false,
		},
		// wstETH/WETH - $96M
		{
			ID:                      [32]byte(common.HexToHash("0xb8fc70e82bc5bb53e773626fcc6a23f7eefa036918d7ef216ecfb1950a94a85e")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
			LoanTokenStr:            "WETH",
			CollateralToken:         common.HexToAddress("0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0"),
			CollateralTokenStr:      "wstETH",
			Oracle:                  common.HexToAddress("0xbD60A6770b27E084E8617335ddE769241B0e71D8"),
			LLTV:                    utils.ParseBigInt("965000000000000000"),
			LoanTokenDecimals:       18,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              true,
			CexOnly:                 false,
		},
		// weETH/WETH - $78M
		{
			ID:                      [32]byte(common.HexToHash("0x37e7484d642d90f14451f1910ba4b7b8e4c3ccdd0ec28f8b2bdb35479e472ba7")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
			LoanTokenStr:            "WETH",
			CollateralToken:         common.HexToAddress("0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee"),
			CollateralTokenStr:      "weETH",
			Oracle:                  common.HexToAddress("0xbDd2F2D473E8D63d1BFb0185B5bDB8046ca48a72"),
			LLTV:                    utils.ParseBigInt("945000000000000000"),
			LoanTokenDecimals:       18,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              true,
			CexOnly:                 false,
		},
		// wstETH/USDC - $60M
		{
			ID:                      [32]byte(common.HexToHash("0xb323495f7e4148be5643a4ea4a8221eef163e4bccfdedc2a6f4696baacbc86cc")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0"),
			CollateralTokenStr:      "wstETH",
			Oracle:                  common.HexToAddress("0x48F7E36EB6B826B2dF4B2E630B62Cd25e89E40e2"),
			LLTV:                    utils.ParseBigInt("860000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 false,
		},
		// WBTC/USDC - $112M
		{
			ID:                      [32]byte(common.HexToHash("0x3a85e619751152991742810df6ec69ce473daef99e28a64ab2340d7b7ccfee49")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599"),
			CollateralTokenStr:      "WBTC",
			Oracle:                  common.HexToAddress("0xDddd770BADd886dF3864029e4B377B5F6a2B6b83"),
			LLTV:                    utils.ParseBigInt("860000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 8,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 true,
		},
		// sUSDS/USDT
		{
			ID:                      [32]byte(common.HexToHash("0x3274643db77a064abd3bc851de77556a4ad2e2f502f4f0c80845fa8f909ecf0b")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
			LoanTokenStr:            "USDT",
			CollateralToken:         common.HexToAddress("0xa3931d71877C0E7a3148CB7Eb4463524FEc27fbD"),
			CollateralTokenStr:      "sUSDS",
			Oracle:                  common.HexToAddress("0x0C426d174FC88B7A25d59945Ab2F7274Bf7B4C79"),
			LLTV:                    utils.ParseBigInt("965000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 false,
		},

		// sUSDe/PYUSD
		{
			ID:                      [32]byte(common.HexToHash("0x90ef0c5a0dc7c4de4ad4585002d44e9d411d212d2f6258e94948beecf8b4c0d5")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0x6c3ea9036406852006290770BEdFcAbA0e23A0e8"),
			LoanTokenStr:            "PYUSD",
			CollateralToken:         common.HexToAddress("0x9D39A5DE30e57443BfF2A8307A4256c8797A3497"),
			CollateralTokenStr:      "sUSDe",
			Oracle:                  common.HexToAddress("0xE6212D05cB5aF3C821Fef1C1A233a678724F9E7E"),
			LLTV:                    utils.ParseBigInt("915000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 false,
		},

		// sdeUSD/USDC
		{
			ID:                      [32]byte(common.HexToHash("0x0f9563442d64ab3bd3bcb27058db0b0d4046a4c46f0acd811dacae9551d2b129")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0x5C5b196aBE0d54485975D1Ec29617D42D9198326"),
			CollateralTokenStr:      "sdeUSD",
			Oracle:                  common.HexToAddress("0x65F9f6d537C2D628D1c2663896436817440eDB72"),
			LLTV:                    utils.ParseBigInt("915000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 false,
		},

		// syrupUSDC/RLUSD
		{
			ID:                      [32]byte(common.HexToHash("0xc0ae375fd761ff19b3f04de5534c0f1ec110f80e1c2ede27c42c1c43c3040394")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0x8292Bb45bf1Ee4d140127049757C2E0fF06317eD"),
			LoanTokenStr:            "RLUSD",
			CollateralToken:         common.HexToAddress("0x80ac24aA929eaF5013f6436cdA2a7ba190f5Cc0b"),
			CollateralTokenStr:      "syrupUSDC",
			Oracle:                  common.HexToAddress("0xf766F4F1Bcb0CBBF4EEF5E26FF7c7f66a713c1B5"),
			LLTV:                    utils.ParseBigInt("915000000000000000"),
			LoanTokenDecimals:       18,
			CollateralTokenDecimals: 6,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 false,
		},

		// wsrUSD/USDC
		{
			ID:                      [32]byte(common.HexToHash("0x1590cb22d797e226df92ebc6e0153427e207299916e7e4e53461389ad68272fb")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0xd3fD63209FA2D55B07A0f6db36C2f43900be3094"),
			CollateralTokenStr:      "wsrUSD",
			Oracle:                  common.HexToAddress("0x938D2eDb20425cF80F008E7ec314Eb456940Da15"),
			LLTV:                    utils.ParseBigInt("945000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 false,
		}}
)
