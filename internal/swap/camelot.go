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

// ---------------------------------------------------------------------------
// Adresses Camelot – Arbitrum One
// ---------------------------------------------------------------------------

var (
	// AMMv2 — UniswapV2-compatible, fee directement dans les pools (1% ou 0.3%)
	CamelotRouterV2 = common.HexToAddress("0xc873fEcbd354f5A56E00E710B90EF4201db2448d")

	// AMMv3 — Algebra concentré (fork Uni V3 sans fee tier fixe)
	CamelotQuoterV3 = common.HexToAddress("0x0Fc73040b26E9bC8514fA028D998E73A254Fa76E")

	// AMMv4 — Algebra v4 (dernière génération Camelot)
	CamelotQuoterV4 = common.HexToAddress("0xFe24b2cDfF01B644995bc248bA8497467d688F7B")
)

// ---------------------------------------------------------------------------
// Signatures ABI
// ---------------------------------------------------------------------------

// AMMv2 : getAmountsOut(uint256 amountIn, address[] path) → uint256[]
var FuncCamelotGetAmountsOut = w3.MustNewFunc(
	"getAmountsOut(uint256,address[])",
	"uint256[]",
)

// AMMv3/v4 Algebra : quoteExactInputSingle(address tokenIn, address tokenOut, uint256 amountIn, uint160 limitSqrtPrice) → uint256 amountOut
// (pas de fee tier — Algebra le détermine dynamiquement)
var FuncAlgebraQuoteExactInputSingle = w3.MustNewFunc(
	"quoteExactInputSingle(address,address,uint256,uint160)",
	"uint256",
)

// ---------------------------------------------------------------------------
// CamelotQuoter — point d'entrée principal (miroir de UniQuoter / AerodromeQuoter)
// ---------------------------------------------------------------------------

// CamelotQuoter cherche le meilleur amountIn acceptable via recherche binaire,
// en testant AMMv2, AMMv3 et AMMv4 pour chaque midpoint.
func CamelotQuoter(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	amountIn *big.Int,
	oraclePrice *big.Int,
	rateLimit time.Duration,
) (lqtypes.PoolEdge, bool) {
	maxSlippage := marketp.MaxSlippage()
	fmt.Printf("camelot: max slippage for %s is: %f\n", marketp.GetPair(), maxSlippage)

	lo := big.NewInt(1)
	hi := new(big.Int).Set(amountIn)
	var best lqtypes.PoolEdge
	found := false

	for i := 0; i < 12 && lo.Cmp(hi) <= 0; i++ {
		mid := new(big.Int).Rsh(new(big.Int).Add(lo, hi), 1)

		result, ok := CamelotQuote(conn, marketp, mid, oraclePrice, rateLimit)
		if ok {
			best = result
			found = true
			lo = new(big.Int).Add(mid, big.NewInt(1))
		} else {
			hi = new(big.Int).Sub(mid, big.NewInt(1))
		}
	}

	if !found {
		fmt.Printf("camelot: no acceptable slippage found for %s\n", marketp.GetPair())
		return lqtypes.PoolEdge{}, false
	}

	fmt.Printf("camelot: acceptable slippage found for %s  %.4f%%\n", marketp.GetPair(), best.WCSlippage)
	best.DexName = "CAMELOT"
	return best, true
}

// CamelotQuote interroge les trois AMMs Camelot pour un amountIn donné
// et retourne le meilleur résultat dans le budget de slippage.
func CamelotQuote(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	amountIn *big.Int,
	oraclePrice *big.Int,
	rateLimit time.Duration,
) (lqtypes.PoolEdge, bool) {
	var best lqtypes.PoolEdge
	found := false

	// --- AMMv2 ---
	time.Sleep(rateLimit)
	if result, ok := camelotV2QuoteCall(conn, marketp, amountIn, oraclePrice); ok {
		fmt.Printf("camelot v2 slippage=%.4f%%\n", result.WCSlippage)
		if result.WCSlippage <= marketp.MaxSlippage() {
			if !found || result.WCAmountOut.Cmp(best.WCAmountOut) > 0 {
				best = result
				found = true
			}
		}
	}

	// --- AMMv3 (Algebra v1.9) ---
	time.Sleep(rateLimit)
	if result, ok := camelotAlgebraQuoteCall(conn, marketp, CamelotQuoterV3, amountIn, oraclePrice, "v3"); ok {
		fmt.Printf("camelot v3 slippage=%.4f%%\n", result.WCSlippage)
		if result.WCSlippage <= marketp.MaxSlippage() {
			if !found || result.WCAmountOut.Cmp(best.WCAmountOut) > 0 {
				best = result
				found = true
			}
		}
	}

	// --- AMMv4 (Algebra v4) ---
	time.Sleep(rateLimit)
	if result, ok := camelotAlgebraQuoteCall(conn, marketp, CamelotQuoterV4, amountIn, oraclePrice, "v4"); ok {
		fmt.Printf("camelot v4 slippage=%.4f%%\n", result.WCSlippage)
		if result.WCSlippage <= marketp.MaxSlippage() {
			if !found || result.WCAmountOut.Cmp(best.WCAmountOut) > 0 {
				best = result
				found = true
			}
		}
	}

	return best, found
}

// ---------------------------------------------------------------------------
// Appels RPC bas niveau
// ---------------------------------------------------------------------------

// camelotV2QuoteCall — AMMv2 UniswapV2-compatible.
// getAmountsOut(uint256, address[]) → uint256[]
// Le path est direct : [collateral, loan].
func camelotV2QuoteCall(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	amountIn *big.Int,
	oraclePrice *big.Int,
) (lqtypes.PoolEdge, bool) {
	path := []common.Address{
		marketp.GetCollateralToken(),
		marketp.GetLoanToken(),
	}

	var amounts []*big.Int

	if err := conn.EthCallCtx(
		context.Background(),
		[]w3types.RPCCaller{
			eth.CallFunc(CamelotRouterV2, FuncCamelotGetAmountsOut, amountIn, path).
				Returns(&amounts),
		},
	); err != nil || len(amounts) == 0 {
		return lqtypes.PoolEdge{}, false
	}

	amountOut := amounts[len(amounts)-1]

	return lqtypes.PoolEdge{
		TokenIn:     marketp.GetCollateralToken(),
		TokenOut:    marketp.GetLoanToken(),
		Router:      CamelotRouterV2,
		WCSlippage:  ComputeSlippage(amountIn, amountOut, oraclePrice),
		WCAmountIn:  new(big.Int).Set(amountIn),
		WCAmountOut: amountOut,
		// offset du champ amountIn dans le calldata AMMv2 :
		// selector(4) + amountIn(32) → offset = 4
		AmountInOffset: 4,
		CalibratedAt:   time.Now(),
	}, true
}

// camelotAlgebraQuoteCall — AMMv3 / AMMv4 (Algebra).
// quoteExactInputSingle(tokenIn, tokenOut, amountIn, sqrtPriceLimitX96) → amountOut
// Algebra n'a pas de fee tier explicite dans le quoter.
func camelotAlgebraQuoteCall(
	conn lqtypes.EthCaller,
	marketp lqtypes.MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	label string,
) (lqtypes.PoolEdge, bool) {
	tokenIn := marketp.GetCollateralToken()
	tokenOut := marketp.GetLoanToken()
	sqrtPriceLimit := big.NewInt(0)

	var amountOut *big.Int

	if err := conn.EthCallCtx(
		context.Background(),
		[]w3types.RPCCaller{
			eth.CallFunc(
				quoterAddr,
				FuncAlgebraQuoteExactInputSingle,
				tokenIn,
				tokenOut,
				amountIn,
				sqrtPriceLimit,
			).Returns(&amountOut),
		},
	); err != nil {
		return lqtypes.PoolEdge{}, false
	}

	if amountOut == nil || amountOut.Sign() == 0 {
		return lqtypes.PoolEdge{}, false
	}

	return lqtypes.PoolEdge{
		TokenIn:     tokenIn,
		TokenOut:    tokenOut,
		Router:      quoterAddr,
		WCSlippage:  ComputeSlippage(amountIn, amountOut, oraclePrice),
		WCAmountIn:  new(big.Int).Set(amountIn),
		WCAmountOut: amountOut,
		// offset amountIn dans quoteExactInputSingle :
		// selector(4) + tokenIn(32) + tokenOut(32) + amountIn(32) → offset = 68
		AmountInOffset: 68,
		CalibratedAt:   time.Now(),
	}, true
}
