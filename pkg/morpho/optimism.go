package morpho

import (
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/ethereum/go-ethereum/common"
)

var (
	OPTIMISM_MARKETS = []MarketParams{
		{
			ID:                      [32]byte(common.HexToHash("0x8e77af0efaf4a3d59f37126c77f6f0ee7b56bcb1363c0986b8f6087b93ba833e")),
			ChainID:                 10,
			LoanToken:               common.HexToAddress("0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85"),
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0x68f180fcCe6836688e9084f035309E29Bf0A2095"),
			CollateralTokenStr:      "WBTC",
			Oracle:                  common.HexToAddress("0x78900c74Cc6B50819f19A04Acd561e94606137AC"),
			LLTV:                    utils.ParseBigInt("860000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 8,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 false,
		},
		{
			ID:                      [32]byte(common.HexToHash("0x80aa1cba2c5907533b5f4e454786c9628ffe8e4ed1a9edcf139325c0fcf09d01")),
			ChainID:                 10,
			LoanToken:               common.HexToAddress("0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85"),
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0xCF9326e24EBfFBEF22ce1050007A43A3c0B6DB55"),
			CollateralTokenStr:      "sUSDC",
			Oracle:                  common.HexToAddress("0x900a3dFDcB3b8b72a57267711dA6F8e8d19B363F"),
			LLTV:                    utils.ParseBigInt("945000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              true,
			CexOnly:                 false,
		},
		{
			ID:                      [32]byte(common.HexToHash("0x173b66359f0741b1c7f1963075cd271c739b6dc73b658e108a54ce6febeb279b")),
			ChainID:                 10,
			LoanToken:               common.HexToAddress("0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85"),
			LoanTokenStr:            "USDC",
			CollateralToken:         common.HexToAddress("0x4200000000000000000000000000000000000006"),
			CollateralTokenStr:      "WETH",
			Oracle:                  common.HexToAddress("0x6475585f7811d03063bF6817306D75F9D9e61735"),
			LLTV:                    utils.ParseBigInt("860000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
			PoolFee:                 500,
			Correlated:              false,
			CexOnly:                 false,
		},
	}
)
