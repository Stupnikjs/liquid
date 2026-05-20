package swap

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

type AeroRoute struct {
	From    common.Address
	To      common.Address
	Stable  bool
	Factory common.Address
}

// aeroRouteABI est la version ABI-compatible de AeroRoute.
// Les noms de champs doivent correspondre exactement aux noms ABI du contrat.
type aeroRouteABI struct {
	From    common.Address `abi:"from"`
	To      common.Address `abi:"to"`
	Stable  bool           `abi:"stable"`
	Factory common.Address `abi:"factory"`
}

var (
	AeroFactory = common.HexToAddress("0x420DD381b31aEf6683db6B902084cB0FFECe40Da")
	AeroRouter  = common.HexToAddress("0xcF77a3Ba9A5CA399B7c97c74d54e5b1Beb874E43")

	// Sélecteur de getAmountsOut(uint256,(address,address,bool,address)[])
	aeroGetAmountsOutSel = crypto.Keccak256(
		[]byte("getAmountsOut(uint256,(address,address,bool,address)[])"),
	)[:4]

	// Types ABI réutilisables
	aeroUint256Type, _ = abi.NewType("uint256", "", nil)
	aeroRouteType, _   = abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "from", Type: "address"},
		{Name: "to", Type: "address"},
		{Name: "stable", Type: "bool"},
		{Name: "factory", Type: "address"},
	})
	aeroUint256SliceType, _ = abi.NewType("uint256[]", "", nil)

	aeroInputArgs = abi.Arguments{
		{Name: "amountIn", Type: aeroUint256Type},
		{Name: "routes", Type: aeroRouteType},
	}
	aeroOutputArgs = abi.Arguments{
		{Name: "amounts", Type: aeroUint256SliceType},
	}
)

// buildAeroCalldata encode manuellement le calldata pour getAmountsOut.
// w3.MustNewFunc ne supporte pas les tuples struct Go → on passe par go-ethereum/accounts/abi.
func buildAeroCalldata(amountIn *big.Int, routes []AeroRoute) ([]byte, error) {
	abiRoutes := make([]aeroRouteABI, len(routes))
	for i, r := range routes {
		abiRoutes[i] = aeroRouteABI{From: r.From, To: r.To, Stable: r.Stable, Factory: r.Factory}
	}

	encoded, err := aeroInputArgs.Pack(amountIn, abiRoutes)
	if err != nil {
		return nil, fmt.Errorf("aero: pack calldata: %w", err)
	}
	return append(aeroGetAmountsOutSel, encoded...), nil
}

// ---------------------------------------------------------------------------

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

		result, ok := aeroAMMQuoteCall(conn, marketp, amountIn, oraclePrice, route)
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

	calldata, err := buildAeroCalldata(amountIn, routes)
	if err != nil {
		fmt.Printf("aero: encode error: %v\n", err)
		return lqtypes.PoolEdge{}, false
	}

	var raw []byte
	msg := &w3types.Message{
		To:    &AeroRouter,
		Input: calldata,
	}
	if err := conn.EthCallCtx(
		context.Background(),
		[]w3types.RPCCaller{
			eth.Call(msg, nil, nil).Returns(&raw),
		},
	); err != nil {
		fmt.Printf("aero: rpc error: %v\n", err)
		return lqtypes.PoolEdge{}, false
	}

	decoded, err := aeroOutputArgs.Unpack(raw)
	if err != nil || len(decoded) == 0 {
		fmt.Printf("aero: decode error: %v\n", err)
		return lqtypes.PoolEdge{}, false
	}

	amounts, ok := decoded[0].([]*big.Int)
	if !ok || len(amounts) == 0 {
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
