package morpho

import (
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/ethereum/go-ethereum/common"
)

var ArbitrumParams = []MarketParams{
	// ARBITRUM — ChainID 42161

	// USDC / wstETH — 86% LLTV (~$8.28M borrow)
	{
		ID:                      [32]byte(common.HexToHash("0x33e0c8ab132390822b07e5dc95033cf250c963153320b7ffca73220664da2ea0")),
		ChainID:                 42161,
		LoanToken:               common.HexToAddress("0xaf88d065e77c8cC2239327C5EDb3A432268e5831"),
		LoanTokenStr:            "USDC",
		CollateralToken:         common.HexToAddress("0x5979D7b546E38E414F7E9822514be443A4800529"),
		CollateralTokenStr:      "wstETH",
		Oracle:                  common.HexToAddress("0x8e02a9b9Cc29d783b2fCB71C3a72651B591cae31"),
		LLTV:                    utils.ParseBigInt("860000000000000000"),
		LoanTokenDecimals:       6,
		CollateralTokenDecimals: 18,
		PoolFee:                 100,
		Correlated:              true,
	},
	// USDC / WBTC — 86% LLTV (~$2.63M borrow)
	{
		ID:                      [32]byte(common.HexToHash("0xe6392ff19d10454b099d692b58c361ef93e31af34ed1ef78232e07c78fe99169")),
		ChainID:                 42161,
		LoanToken:               common.HexToAddress("0xaf88d065e77c8cC2239327C5EDb3A432268e5831"),
		LoanTokenStr:            "USDC",
		CollateralToken:         common.HexToAddress("0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f"),
		CollateralTokenStr:      "WBTC",
		Oracle:                  common.HexToAddress("0x88193FcB705d29724A40Bb818eCAA47dD5F014d9"),
		LLTV:                    utils.ParseBigInt("860000000000000000"),
		LoanTokenDecimals:       6,
		CollateralTokenDecimals: 8,
		PoolFee:                 500,
		Correlated:              false,
	},
	// USDT0 / WBTC — 86% LLTV (~$1.22M borrow)
	{
		ID:                      [32]byte(common.HexToHash("0xed06d9e82d7c35ca80d3983194e15462a96202bd875800af18183321f4611868")),
		ChainID:                 42161,
		LoanToken:               common.HexToAddress("0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"),
		LoanTokenStr:            "USDT0",
		CollateralToken:         common.HexToAddress("0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f"),
		CollateralTokenStr:      "WBTC",
		Oracle:                  common.HexToAddress("0xEDcAE878827fc68B9bC9c700CA17c20F811b1612"),
		LLTV:                    utils.ParseBigInt("860000000000000000"),
		LoanTokenDecimals:       6,
		CollateralTokenDecimals: 8,
		PoolFee:                 500,
		Correlated:              false,
	},
	// USDT0 / wstETH — 86% LLTV (~$357k borrow)
	{
		ID:                      [32]byte(common.HexToHash("0x209fa1520640f664f59f7c1f955d52e8b81ead826edf439b48254d21d24b97a9")),
		ChainID:                 42161,
		LoanToken:               common.HexToAddress("0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"),
		LoanTokenStr:            "USDT0",
		CollateralToken:         common.HexToAddress("0x5979D7b546E38E414F7E9822514be443A4800529"),
		CollateralTokenStr:      "wstETH",
		Oracle:                  common.HexToAddress("0x979e4C611e4da2776404fE9346D77e95B01BfD82"),
		LLTV:                    utils.ParseBigInt("860000000000000000"),
		LoanTokenDecimals:       6,
		CollateralTokenDecimals: 18,
		PoolFee:                 100,
		Correlated:              true,
	},
}
