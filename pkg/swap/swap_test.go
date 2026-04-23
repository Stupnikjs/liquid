package swap

import (
	"log"
	"math/big"
	"os"
	"testing"

	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/joho/godotenv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

// setupClient crée un client w3 connecté au RPC défini dans l'env.
// Lance t.Skip si INTEGRATION_RPC_URL n'est pas défini.
func setupClient(t *testing.T) *w3.Client {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	t.Helper()
	rpcURL := os.Getenv("INTEGRATION_RPC_URL")
	if rpcURL == "" {
		t.Skip("INTEGRATION_RPC_URL non défini, test d'intégration ignoré")
	}
	client, err := w3.Dial(rpcURL)
	if err != nil {
		t.Fatalf("impossible de se connecter au RPC: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// marketParams de référence — Base mainnet.
func testMarketParams() morpho.MarketParams {
	return morpho.MarketParams{
		// WETH comme collateral, USDC comme loan (Base mainnet)
		CollateralToken:    common.HexToAddress("0x4200000000000000000000000000000000000006"), // WETH Base
		LoanToken:          common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"), // USDC Base
		CollateralTokenStr: "WETH",
		LoanTokenStr:       "USDC",
		LLTV:               new(big.Int).Mul(big.NewInt(945), new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil)), // 94.5% = 945e15
	}
}

// Uniswap QuoterV2 sur Base mainnet
var uniswapQuoterAddr = common.HexToAddress("0x3d4e44Eb1374240CE5F1B136cf68A4f7f7B5822F")

// oraclePrice simulé : 1 WETH = 2000 USDC, exprimé selon la convention Morpho.
// USDC a 6 décimales, WETH 18 décimales.
// oraclePrice = price * 1e36 / 10^(decimalsLoan - decimalsCollateral)
//
//	= 2000 * 1e36 * 1e18 / 1e6 = 2000 * 1e48... non.
//
// Convention Morpho : oraclePrice = collateralPriceInLoan * 10^(36 + decimalsLoan - decimalsCollateral)
//
//	= 2000 * 10^(36 + 6 - 18) = 2000 * 10^24
func testOraclePrice() *big.Int {
	p := new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)
	p.Mul(p, big.NewInt(2000))
	return p
}

// -----------------------------------------------------------------------------
// Test : MaxSlippage
// -----------------------------------------------------------------------------

func TestMaxSlippage_Unit(t *testing.T) {
	cases := []struct {
		name    string
		lltvWei *big.Int
		wantGt  float64
		wantLt  float64
	}{
		{
			name:    "LLTV 94.5%",
			lltvWei: new(big.Int).Mul(big.NewInt(945), new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil)),
			wantGt:  5.0,
			wantLt:  6.0,
		},
		{
			name:    "LLTV 80%",
			lltvWei: new(big.Int).Mul(big.NewInt(8), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)),
			wantGt:  19.0,
			wantLt:  21.0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MaxSlippage(tc.lltvWei)
			if got <= tc.wantGt || got >= tc.wantLt {
				t.Errorf("MaxSlippage(%s) = %f, attendu entre %f et %f",
					tc.lltvWei.String(), got, tc.wantGt, tc.wantLt)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Test : computeSlippage (via Quote avec oraclePrice=0)
// -----------------------------------------------------------------------------

func TestComputeSlippage_ZeroOraclePrice(t *testing.T) {
	client := setupClient(t)
	mp := testMarketParams()

	amountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil) // 0.1 WETH
	zeroPrice := big.NewInt(0)

	result, err := Quote(client, mp, uniswapQuoterAddr, amountIn, zeroPrice)
	if err != nil {
		t.Logf("Quote avec oraclePrice=0 retourne une erreur (attendu): %v", err)
		return
	}
	if result != nil {
		t.Logf("slippage avec oraclePrice=0: %f%%", result.Slippage)
	}
}

// -----------------------------------------------------------------------------
// Test : Quote — intégration RPC Base
// -----------------------------------------------------------------------------

func TestQuote_SmallAmount(t *testing.T) {
	client := setupClient(t)
	mp := testMarketParams()
	oraclePrice := testOraclePrice()

	// 0.001 WETH → slippage quasi nul attendu sur Base (liquidité profonde)
	amountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil)

	result, err := Quote(client, mp, uniswapQuoterAddr, amountIn, oraclePrice)
	if err != nil {
		t.Fatalf("Quote échoué pour un petit montant: %v", err)
	}
	if result == nil {
		t.Fatal("Quote retourne nil sans erreur")
	}

	t.Logf("AmountIn:  %s wei WETH", result.AmountIn.String())
	t.Logf("AmountOut: %s unités USDC (6 dec)", result.AmountOut.String())
	t.Logf("Fee:       %d", result.Fee)
	t.Logf("Slippage:  %.4f%%", result.Slippage)

	maxSlip := MaxSlippage(mp.LLTV)
	if result.Slippage > maxSlip {
		t.Errorf("slippage %f%% dépasse le max autorisé %f%%", result.Slippage, maxSlip)
	}
	if result.AmountOut.Sign() <= 0 {
		t.Error("AmountOut devrait être positif")
	}
}

func TestQuote_LargeAmount_ExpectErrorOrReducedAmount(t *testing.T) {
	client := setupClient(t)
	mp := testMarketParams()
	oraclePrice := testOraclePrice()

	// 1000 WETH — slippage élevé probable même sur Base
	amountIn := new(big.Int).Mul(
		big.NewInt(1000),
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
	)

	result, err := Quote(client, mp, uniswapQuoterAddr, amountIn, oraclePrice)
	if err != nil {
		t.Logf("Aucun slippage acceptable pour 1000 WETH (attendu): %v", err)
		return
	}
	if result.AmountIn.Cmp(amountIn) >= 0 {
		t.Errorf("AmountIn résultant %s devrait être < %s", result.AmountIn, amountIn)
	}
	t.Logf("Montant réduit accepté: %s, slippage: %.4f%%", result.AmountIn, result.Slippage)
}

// -----------------------------------------------------------------------------
// Test : QuoteBinarySearch — intégration RPC Base
// -----------------------------------------------------------------------------

func TestQuoteBinarySearch_FindsMaxAmount(t *testing.T) {
	client := setupClient(t)
	mp := testMarketParams()
	oraclePrice := testOraclePrice()

	// 1 WETH comme point de départ
	amountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

	result, err := QuoteBinarySearch(client, mp, uniswapQuoterAddr, amountIn, oraclePrice)
	if err != nil {
		t.Logf("QuoteBinarySearch: aucun montant valide trouvé: %v", err)
		return
	}
	if result == nil {
		t.Fatal("QuoteBinarySearch retourne nil sans erreur")
	}

	t.Logf("Meilleur montant trouvé: %s", result.AmountIn.String())
	t.Logf("AmountOut:              %s", result.AmountOut.String())
	t.Logf("Slippage:               %.4f%%", result.Slippage)

	maxSlip := MaxSlippage(mp.LLTV)
	if result.Slippage > maxSlip {
		t.Errorf("slippage %f%% dépasse le max %f%%", result.Slippage, maxSlip)
	}
}

func TestQuoteBinarySearch_VsQuote_Consistency(t *testing.T) {
	// QuoteBinarySearch devrait trouver un AmountIn >= celui de Quote (plus optimal)
	client := setupClient(t)
	mp := testMarketParams()
	oraclePrice := testOraclePrice()

	amountIn := new(big.Int).Mul(
		big.NewInt(5),
		new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil), // 0.5 WETH
	)

	resultQuote, errQ := Quote(client, mp, uniswapQuoterAddr, amountIn, oraclePrice)
	resultBinary, errB := QuoteBinarySearch(client, mp, uniswapQuoterAddr, amountIn, oraclePrice)

	if errQ != nil && errB != nil {
		t.Logf("Les deux méthodes échouent: Q=%v B=%v", errQ, errB)
		return
	}

	if resultQuote != nil && resultBinary != nil {
		t.Logf("Quote AmountIn:        %s (slippage %.4f%%)", resultQuote.AmountIn, resultQuote.Slippage)
		t.Logf("BinarySearch AmountIn: %s (slippage %.4f%%)", resultBinary.AmountIn, resultBinary.Slippage)

		if resultBinary.AmountIn.Cmp(resultQuote.AmountIn) < 0 {
			t.Errorf("BinarySearch devrait trouver un montant >= Quote: %s < %s",
				resultBinary.AmountIn, resultQuote.AmountIn)
		}
	}
}
