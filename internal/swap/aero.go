package swap

import (
	"context"
	"fmt"
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

var (
	AeroFactory = common.HexToAddress("0x420DD381b31aEf6683db6B902084cB0FFECe40Da")
	AeroRouter  = common.HexToAddress("0xcF77a3Ba9A5CA399B7c97c74d54e5b1Beb874E43")
)

func AerodromeQuoter(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
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

		result, ok := aeroAMMQuoteCall(
			conn,
			marketp,
			amountIn,
			oraclePrice,
			route,
		)
		if !ok {
			continue
		}

		fmt.Printf("aerodrome quote stable=%v slippage=%.4f%%\n", route[0].Stable, result.WCSlippage)

		if result.WCSlippage <= marketp.MaxSlippage() {
			if !found || result.WCAmountOut.Cmp(best.WCAmountOut) > 0 {
				best = result
				found = true
			}
		}
	}

	if !found {
		fmt.Printf("no acceptable slippage found for %s\n", marketp.GetPair())
		return lqtypes.PoolEdge{}, false
	}

	fmt.Printf("acceptable slippage found for %s  %.4f%%\n", marketp.GetPair(), best.WCSlippage)
	best.AmountInOffset = 36
	best.DexName = "AERO"
	return best, true
}

func aeroAMMQuoteCall(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	amountIn *big.Int,
	oraclePrice *big.Int,
	routes []AeroRoute,
) (lqtypes.PoolEdge, bool) {

	var amounts []*big.Int

	err := conn.EthCallCtx(
		context.Background(),
		[]w3types.RPCCaller{
			eth.CallFunc(
				AeroRouter,
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
		TokenIn:      marketp.GetCollateralToken(),
		TokenOut:     marketp.GetLoanToken(),
		Router:       AeroRouter,
		WCSlippage:   ComputeSlippage(amountIn, amountOut, oraclePrice),
		WCAmountIn:   new(big.Int).Set(amountIn),
		WCAmountOut:  amountOut,
		CalibratedAt: time.Now(),
	}, true
}
