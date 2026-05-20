package swap

import (
	"context"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

/*
struct Route {
    address from;
    address to;
    bool stable;
    address factory;
}
*/

var FuncGetAmountsOut = w3.MustNewFunc(
	"getAmountsOut(uint256,(address,address,bool,address)[])",
	"uint256[]",
)

type AeroRoute struct {
	From    common.Address
	To      common.Address
	Stable  bool
	Factory common.Address
}

var AeroFactory = common.HexToAddress("0x420DD381b31aEf6683db6B902084cB0FFECe40Da") // Sushiswap factory, used by Aerodrome

func AerodromeQuoter(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	rateLimit time.Duration,
) (lqtypes.PoolEdge, bool) {

	var best lqtypes.PoolEdge
	found := false

	routes := [][]AeroRoute{
		{
			{
				From:    marketp.GetCollateralToken(),
				To:      marketp.GetLoanToken(),
				Stable:  false,
				Factory: AeroFactory,
			},
		},
		{
			{
				From:    marketp.GetCollateralToken(),
				To:      marketp.GetLoanToken(),
				Stable:  true,
				Factory: AeroFactory,
			},
		},
	}

	for _, route := range routes {
		time.Sleep(rateLimit)

		result, ok := aeroQuoteCall(
			conn,
			marketp,
			quoterAddr,
			amountIn,
			oraclePrice,
			route,
		)

		if !ok {
			continue
		}

		if result.WCSlippage <= marketp.MaxSlippage() {
			if !found || result.WCAmountOut.Cmp(best.WCAmountOut) > 0 {
				best = result
				found = true
			}
		}
	}

	if !found {
		return lqtypes.PoolEdge{}, false
	}

	// offset ABI si tu fais du patch calldata plus tard
	best.AmountInOffset = 36

	return best, true
}

func aeroQuoteCall(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	router common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	routes []AeroRoute,
) (lqtypes.PoolEdge, bool) {

	var amounts []*big.Int

	err := conn.EthCallCtx(
		context.Background(),
		[]w3types.RPCCaller{
			eth.CallFunc(
				router,
				FuncGetAmountsOut,
				amountIn,
				routes,
			).Returns(&amounts),
		},
	)

	if err != nil || len(amounts) == 0 {
		return lqtypes.PoolEdge{}, false
	}

	amountOut := amounts[len(amounts)-1]

	return lqtypes.PoolEdge{
		TokenIn:  marketp.GetCollateralToken(),
		TokenOut: marketp.GetLoanToken(),
		Router:   router,

		WCSlippage: ComputeSlippage(
			amountIn,
			amountOut,
			oraclePrice,
		),

		WCAmountIn:   amountIn,
		WCAmountOut:  amountOut,
		CalibratedAt: time.Now(),
	}, true
}
