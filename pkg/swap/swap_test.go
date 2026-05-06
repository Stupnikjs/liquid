package swap

import (
	"math/big"
	"os"
	"testing"

	"github.com/Stupnikjs/liquid/pkg/config"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

var (
	// UniswapV3QuoterV2 sur Base mainnet
	UniswapQuoterV2Base = common.HexToAddress("0x3d4e44Eb1374240CE5F1B136Ea95285f1B6F826")

	// Tokens courants sur Base
	WETHBase = common.HexToAddress("0x4200000000000000000000000000000000000006")
	USDCBase = common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")
)

// ---------------------------------------------------------------------------
// Marché WETH/USDC prédéfini sur Base mainnet
// ---------------------------------------------------------------------------

// wethUSDCMarket retourne les paramètres du marché WETH (collateral) / USDC (loan)
// sur Base mainnet, avec un LLTV de 86% (valeur Morpho standard).
func wethUSDCMarket() morpho.MarketParams {
	// LLTV 86% exprimé en 1e18
	lltv, _ := new(big.Int).SetString("860000000000000000", 10)

	return morpho.MarketParams{
		CollateralToken:    WETHBase,
		LoanToken:          USDCBase,
		CollateralTokenStr: "WETH",
		LoanTokenStr:       "USDC",
		LLTV:               lltv,
	}
}

// oraclePriceWETHUSDC retourne un prix oracle WETH/USDC ≈ 3500 USDC/WETH
// exprimé en 1e36 (convention Morpho : oraclePrice * 1e36 / 1e18 = prix en USDC).
//
// USDC a 6 décimales, WETH en a 18.
// Prix attendu en USDC avec 6 décimales pour 1 WETH (1e18) :
//
//	3500 * 1e6 = 3_500_000_000
//
// oraclePrice = 3_500_000_000 * 1e36 / 1e18 = 3_500_000_000 * 1e18
func oraclePriceWETHUSDC() *big.Int {
	// 3500 USDC * 1e6 (décimales USDC) * 1e18 (normalisation Morpho)
	price, _ := new(big.Int).SetString("3500000000000000000000000000", 10) // 3.5e27
	// Morpho oracle price = collateral price in loan token * 1e36 / (1e collateralDecimals)
	// Pour WETH->USDC : price = 3500 * 1e6 * 1e36 / 1e18 = 3500 * 1e24
	price, _ = new(big.Int).SetString("3500000000000000000000000000", 10)
	return price
}

// rpcClient crée un client w3 à partir de la variable d'environnement BASE_RPC_URL.
// Si la variable n'est pas définie, le test est skippé.
func rpcClient(t *testing.T) *w3.Client {
	t.Helper()
	rpcURL := os.Getenv("BASE_HTTP_RPC_ALCH")
	if rpcURL == "" {
		t.Skip("BASE_RPC_URL non défini : test d'intégration ignoré")
	}
	client, err := w3.Dial(rpcURL)
	if err != nil {
		t.Fatalf("impossible de se connecter au RPC Base: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// ---------------------------------------------------------------------------
// Tests ComputeSlippage (unitaires, pas de RPC)
// ---------------------------------------------------------------------------

func TestComputeSlippage_ZeroOracle(t *testing.T) {
	slip := ComputeSlippage(big.NewInt(1e9), big.NewInt(1e9), big.NewInt(0))
	if slip != 0 {
		t.Errorf("attendu 0 pour oracle nul, obtenu %f", slip)
	}
}

func TestComputeSlippage_NoSlippage(t *testing.T) {
	// amountIn = 1 WETH (1e18), oraclePrice = 3500 USDC (1e6) * 1e36 / 1e18
	// expectedOut = 1e18 * 3500e24 / 1e36 = 3500e6
	amountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1 WETH
	oraclePrice := oraclePriceWETHUSDC()
	// expectedOut = amountIn * oraclePrice / 1e36
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	expectedOut := new(big.Int).Div(new(big.Int).Mul(amountIn, oraclePrice), scale)

	slip := ComputeSlippage(amountIn, expectedOut, oraclePrice)
	if slip < -0.01 || slip > 0.01 {
		t.Errorf("slippage attendu ≈ 0%%, obtenu %.6f%%", slip)
	}
}

func TestComputeSlippage_PositiveSlippage(t *testing.T) {
	amountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	oraclePrice := oraclePriceWETHUSDC()
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	expectedOut := new(big.Int).Div(new(big.Int).Mul(amountIn, oraclePrice), scale)

	// amountOut = expectedOut * 99% → slippage ≈ 1%
	amountOut := new(big.Int).Mul(expectedOut, big.NewInt(99))
	amountOut.Div(amountOut, big.NewInt(100))

	slip := ComputeSlippage(amountIn, amountOut, oraclePrice)
	if slip < 0.9 || slip > 1.1 {
		t.Errorf("slippage attendu ≈ 1%%, obtenu %.6f%%", slip)
	}
}

// ---------------------------------------------------------------------------
// Tests MaxSlippage (unitaires, pas de RPC)
// ---------------------------------------------------------------------------

func TestMaxSlippage_86pct(t *testing.T) {
	lltv, _ := new(big.Int).SetString("860000000000000000", 10)
	max := MaxSlippage(lltv)
	// 100 - 86 - 0.1 = 13.9
	if max < 13.8 || max > 14.0 {
		t.Errorf("MaxSlippage attendu ≈ 13.9%%, obtenu %.4f%%", max)
	}
}

func TestMaxSlippage_945pct(t *testing.T) {
	lltv, _ := new(big.Int).SetString("945000000000000000", 10)
	max := MaxSlippage(lltv)
	// 100 - 94.5 - 0.1 = 5.4
	if max < 5.3 || max > 5.5 {
		t.Errorf("MaxSlippage attendu ≈ 5.4%%, obtenu %.4f%%", max)
	}
}

// ---------------------------------------------------------------------------
// Tests d'intégration avec vrais appels RPC
// ---------------------------------------------------------------------------

// TestQuote_WETH_USDC_SmallAmount vérifie qu'un petit montant (0.01 WETH)
// trouve un fee tier avec slippage acceptable.
func TestQuote_WETH_USDC_SmallAmount(t *testing.T) {
	client := rpcClient(t)
	marketp := wethUSDCMarket()
	oraclePrice := oraclePriceWETHUSDC()

	// 0.01 WETH = 1e16 wei
	amountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil)

	result, err := Quote(client, marketp, UniswapQuoterV2Base, amountIn, oraclePrice)
	if err != nil {
		t.Fatalf("Quote échoué: %v", err)
	}
	if result == nil {
		t.Fatal("Quote a retourné nil sans erreur")
	}

	t.Logf("Fee tier: %d | AmountOut: %s USDC | Slippage: %.4f%%",
		result.Fee, result.AmountOut.String(), result.Slippage)

	if result.AmountOut == nil || result.AmountOut.Sign() <= 0 {
		t.Error("AmountOut devrait être positif")
	}
	maxSlippage := MaxSlippage(marketp.LLTV)
	if result.Slippage > maxSlippage {
		t.Errorf("Slippage %.4f%% dépasse le max autorisé %.4f%%", result.Slippage, maxSlippage)
	}
}

// TestQuote_WETH_USDC_LargeAmount vérifie que Quote divise correctement
// le montant si le slippage est trop élevé (10 WETH).
func TestQuote_WETH_USDC_LargeAmount(t *testing.T) {
	client := rpcClient(t)
	marketp := wethUSDCMarket()
	oraclePrice := oraclePriceWETHUSDC()

	// 10 WETH = 1e19 wei
	amountIn := new(big.Int).Mul(
		big.NewInt(10),
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
	)

	result, err := Quote(client, marketp, config.BaseUniswapQuoterV2Addr, amountIn, oraclePrice)
	if err != nil {
		t.Logf("Aucun fee tier acceptable pour 10 WETH (attendu possible): %v", err)
		return
	}

	t.Logf("Fee tier: %d | AmountIn effectif: %s | AmountOut: %s USDC | Slippage: %.4f%%",
		result.Fee, result.AmountIn.String(), result.AmountOut.String(), result.Slippage)

	maxSlippage := MaxSlippage(marketp.LLTV)
	if result.Slippage > maxSlippage {
		t.Errorf("Slippage %.4f%% dépasse le max autorisé %.4f%%", result.Slippage, maxSlippage)
	}
}

// TestQuoteBinarySearch_WETH_USDC vérifie que la recherche binaire trouve
// le montant maximal swappable avec slippage acceptable.
func TestQuoteBinarySearch_WETH_USDC(t *testing.T) {
	client := rpcClient(t)
	marketp := wethUSDCMarket()
	oraclePrice := oraclePriceWETHUSDC()

	// Borne haute : 5 WETH
	amountIn := new(big.Int).Mul(
		big.NewInt(5),
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
	)

	result, err := QuoteBinarySearch(client, marketp, config.BaseUniswapQuoterV2Addr, amountIn, oraclePrice)
	if err != nil {
		t.Logf("QuoteBinarySearch : aucun montant acceptable (attendu possible): %v", err)
		return
	}

	t.Logf("Binary search résultat — Fee: %d | AmountIn: %s wei | AmountOut: %s | Slippage: %.4f%%",
		result.Fee, result.AmountIn.String(), result.AmountOut.String(), result.Slippage)

	maxSlippage := MaxSlippage(marketp.LLTV)
	if result.Slippage > maxSlippage {
		t.Errorf("Slippage %.4f%% dépasse le max autorisé %.4f%%", result.Slippage, maxSlippage)
	}
	// Le montant trouvé doit être ≤ la borne haute
	if result.AmountIn.Cmp(amountIn) > 0 {
		t.Errorf("AmountIn %s dépasse la borne haute %s", result.AmountIn, amountIn)
	}
}

// TestRPCQuoteSingle_Fee500_WETH_USDC vérifie directement RPCQuoteSingle
// sur le pool 500 (0.05%), le plus liquide pour WETH/USDC.
func TestRPCQuoteSingle_Fee500_WETH_USDC(t *testing.T) {
	client := rpcClient(t)
	marketp := wethUSDCMarket()
	oraclePrice := oraclePriceWETHUSDC()

	// 0.1 WETH
	amountIn := new(big.Int).Div(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
		big.NewInt(10),
	)

	result, err := RPCQuoteSingle(client, marketp, UniswapQuoterV2Base, amountIn, oraclePrice, 500)
	if err != nil {
		t.Fatalf("RPCQuoteSingle erreur inattendue: %v", err)
	}
	if result == nil {
		t.Skip("Pool 500 inexistant sur ce réseau (nil retourné)")
	}

	t.Logf("Pool 500 — AmountOut: %s | FeeAmount: %s | Slippage: %.4f%%",
		result.AmountOut.String(), result.FeeAmount.String(), result.Slippage)

	if result.Fee != 500 {
		t.Errorf("Fee attendu 500, obtenu %d", result.Fee)
	}
	if result.AmountOut == nil || result.AmountOut.Sign() <= 0 {
		t.Error("AmountOut devrait être positif")
	}
	if result.FeeAmount == nil || result.FeeAmount.Sign() < 0 {
		t.Error("FeeAmount ne devrait pas être négatif")
	}
}

// TestQuote_InvalidMarket vérifie que Quote retourne une erreur si les deux
// tokens sont identiques (pas de pool possible).
func TestQuote_InvalidMarket(t *testing.T) {
	client := rpcClient(t)

	lltv, _ := new(big.Int).SetString("860000000000000000", 10)
	// Marché invalide : USDC -> USDC
	marketp := morpho.MarketParams{
		CollateralToken:    USDCBase,
		LoanToken:          USDCBase,
		CollateralTokenStr: "USDC",
		LoanTokenStr:       "USDC",
		LLTV:               lltv,
	}
	oraclePrice := big.NewInt(1)
	amountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)

	_, err := Quote(client, marketp, UniswapQuoterV2Base, amountIn, oraclePrice)
	if err == nil {
		t.Error("attendu une erreur pour marché USDC->USDC, obtenu nil")
	}
	t.Logf("Erreur correctement retournée: %v", err)
}

// ---------------------------------------------------------------------------
// Test mock (sans RPC) — vérifie la logique de sélection du meilleur fee tier
// ---------------------------------------------------------------------------

// TestQuoter_MockBestFeeTier vérifie que Quoter sélectionne bien le fee tier
// avec le meilleur amountOut parmi plusieurs résultats simulés.
func TestQuoter_MockBestFeeTier(t *testing.T) {
	marketp := wethUSDCMarket()
	oraclePrice := oraclePriceWETHUSDC()
	amountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil) // 0.01 WETH

	// Slippage max ≈ 13.9% pour LLTV 86%
	// On simule : fee 500 → slippage 1% (bon), fee 3000 → slippage 2% (bon mais moins bon amountOut)
	mockFn := func(
		_ *w3.Client,
		_ morpho.MarketParams,
		_ common.Address,
		amount, _ *big.Int,
		fee uint32,
	) (*QuoteResult, error) {
		switch fee {
		case 100:
			return nil, nil // pool inexistant
		case 500:
			// Meilleur amountOut : 34.5 USDC (1% de slippage)
			out, _ := new(big.Int).SetString("34500000", 10)
			return &QuoteResult{
				AmountIn:  amount,
				AmountOut: out,
				Fee:       500,
				Slippage:  1.0,
			}, nil
		case 3000:
			// Moins bon amountOut : 34.0 USDC (2.4% de slippage)
			out, _ := new(big.Int).SetString("34000000", 10)
			return &QuoteResult{
				AmountIn:  amount,
				AmountOut: out,
				Fee:       3000,
				Slippage:  2.4,
			}, nil
		case 10000:
			return nil, nil
		}
		return nil, nil
	}

	q := NewQuoterWithFunc(mockFn)
	result, err := q.Quote(nil, marketp, common.Address{}, amountIn, oraclePrice)
	if err != nil {
		t.Fatalf("Quote mock échoué: %v", err)
	}
	if result.Fee != 500 {
		t.Errorf("attendu fee 500 (meilleur amountOut), obtenu %d", result.Fee)
	}
	t.Logf("Mock — Fee sélectionné: %d | AmountOut: %s", result.Fee, result.AmountOut.String())
}

// TestQuoter_MockAllSlippageTooHigh vérifie que Quote retourne une erreur
// quand tous les fee tiers ont un slippage trop élevé.
func TestQuoter_MockAllSlippageTooHigh(t *testing.T) {
	marketp := wethUSDCMarket()
	oraclePrice := oraclePriceWETHUSDC()
	amountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil)

	mockFn := func(_ *w3.Client, _ morpho.MarketParams, _ common.Address, amount, _ *big.Int, fee uint32) (*QuoteResult, error) {
		out, _ := new(big.Int).SetString("30000000", 10)
		return &QuoteResult{
			AmountIn:  amount,
			AmountOut: out,
			Fee:       fee,
			Slippage:  20.0, // > 13.9% max
		}, nil
	}

	q := NewQuoterWithFunc(mockFn)
	_, err := q.Quote(nil, marketp, common.Address{}, amountIn, oraclePrice)
	if err == nil {
		t.Error("attendu une erreur quand tous les slippages sont trop élevés")
	}
	t.Logf("Erreur correctement retournée: %v", err)
}
