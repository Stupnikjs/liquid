package swap

import (
	"math/big"
	"testing"
	"time"

	"github.com/Stupnikjs/liquid/internal/connector"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
)

// ---------------------------------------------------------------------------
// Helpers de test
// ---------------------------------------------------------------------------

// newMarketParams construit un MarketParams minimal pour les tests.
func newMarketParams(lltv int64) morpho.MarketParams {
	// LLTV exprimé en 1e18, ex: 0.9 → 900000000000000000
	lltvWei := new(big.Int).Mul(big.NewInt(lltv), new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil))
	return morpho.MarketParams{
		CollateralToken:    common.HexToAddress("0x1111111111111111111111111111111111111111"),
		LoanToken:          common.HexToAddress("0x2222222222222222222222222222222222222222"),
		CollateralTokenStr: "WETH",
		LoanTokenStr:       "USDC",
		LLTV:               lltvWei,
	}
}

// mockQuoteSingle retourne un PoolEdge fixe avec le slippage et amountOut fournis.
func mockQuoteSingle(slippage float64, amountOut *big.Int) SingleQuoterFunc {
	return func(
		conn *connector.Connector,
		marketp morpho.MarketParams,
		uniswapQuoterAddr common.Address,
		amountIn, oraclePrice *big.Int,
		fee uint32,
	) (PoolEdge, bool) {
		if amountOut == nil {
			return PoolEdge{}, false
		}
		return PoolEdge{
			TokenIn:      marketp.CollateralToken,
			TokenOut:     marketp.LoanToken,
			Router:       uniswapQuoterAddr,
			Fee:          fee,
			WCSlippage:   slippage,
			WCAmountIn:   new(big.Int).Set(amountIn),
			WCAmountOut:  new(big.Int).Set(amountOut),
			CalibratedAt: time.Now(),
		}, true
	}
}

// mockNoPool simule un DEX sans pool disponible.
func mockNoPool() SingleQuoterFunc {
	return func(_ *connector.Connector, _ morpho.MarketParams, _ common.Address, _, _ *big.Int, _ uint32) (PoolEdge, bool) {
		return PoolEdge{}, false
	}
}

// mockByFee retourne des résultats différents selon le fee tier.
func mockByFee(results map[uint32]struct {
	slippage  float64
	amountOut *big.Int
}) SingleQuoterFunc {
	return func(
		_ *connector.Connector,
		marketp morpho.MarketParams,
		addr common.Address,
		amountIn, _ *big.Int,
		fee uint32,
	) (PoolEdge, bool) {
		r, ok := results[fee]
		if !ok || r.amountOut == nil {
			return PoolEdge{}, false
		}
		return PoolEdge{
			TokenIn:      marketp.CollateralToken,
			TokenOut:     marketp.LoanToken,
			Router:       addr,
			Fee:          fee,
			WCSlippage:   r.slippage,
			WCAmountIn:   new(big.Int).Set(amountIn),
			WCAmountOut:  new(big.Int).Set(r.amountOut),
			CalibratedAt: time.Now(),
		}, true
	}
}

var (
	zeroAddr        = common.Address{}
	oneEth          = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1e18
	oraclePrice1e36 = new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil) // 1:1 ratio
)

// ---------------------------------------------------------------------------
// ComputeSlippage
// ---------------------------------------------------------------------------

func TestComputeSlippage_ZeroOraclePrice(t *testing.T) {
	slip := ComputeSlippage(oneEth, oneEth, big.NewInt(0))
	if slip != 0 {
		t.Errorf("expected 0 slippage with zero oracle price, got %f", slip)
	}
}

func TestComputeSlippage_NilOraclePrice(t *testing.T) {
	slip := ComputeSlippage(oneEth, oneEth, nil)
	if slip != 0 {
		t.Errorf("expected 0 slippage with nil oracle price, got %f", slip)
	}
}

func TestComputeSlippage_NoSlippage(t *testing.T) {
	// amountIn == amountOut, ratio oracle 1:1 → slippage = 0
	slip := ComputeSlippage(oneEth, oneEth, oraclePrice1e36)
	if slip != 0 {
		t.Errorf("expected 0%% slippage, got %.4f%%", slip)
	}
}

func TestComputeSlippage_HalfOut(t *testing.T) {
	// amountOut = amountIn / 2 → slippage = 50%
	halfEth := new(big.Int).Div(oneEth, big.NewInt(2))
	slip := ComputeSlippage(oneEth, halfEth, oraclePrice1e36)
	if slip < 49.9 || slip > 50.1 {
		t.Errorf("expected ~50%% slippage, got %.4f%%", slip)
	}
}

func TestComputeSlippage_Positive(t *testing.T) {
	// amountOut légèrement inférieur → slippage > 0
	amountOut := new(big.Int).Sub(oneEth, big.NewInt(1e14)) // oneEth - 0.01%
	slip := ComputeSlippage(oneEth, amountOut, oraclePrice1e36)
	if slip <= 0 {
		t.Errorf("expected positive slippage, got %.6f%%", slip)
	}
}

// ---------------------------------------------------------------------------
// MaxSlippage
// ---------------------------------------------------------------------------

func TestMaxSlippage_90LLTV(t *testing.T) {
	// LLTV = 90% → bonus = 10% → maxSlippage = 9.9%
	lltv := new(big.Int).Mul(big.NewInt(90), new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil))
	ms := MaxSlippage(lltv)
	expected := 9.9
	if ms < expected-0.01 || ms > expected+0.01 {
		t.Errorf("expected MaxSlippage ~%.2f%%, got %.4f%%", expected, ms)
	}
}

func TestMaxSlippage_77LLTV(t *testing.T) {
	// LLTV = 77% → bonus = 23% → maxSlippage = 22.9%
	lltv := new(big.Int).Mul(big.NewInt(77), new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil))
	ms := MaxSlippage(lltv)
	expected := 22.9
	if ms < expected-0.01 || ms > expected+0.01 {
		t.Errorf("expected MaxSlippage ~%.2f%%, got %.4f%%", expected, ms)
	}
}

// ---------------------------------------------------------------------------
// PoolEdge.IsStale
// ---------------------------------------------------------------------------

func TestIsStale_Fresh(t *testing.T) {
	p := PoolEdge{CalibratedAt: time.Now()}
	if p.IsStale(1 * time.Minute) {
		t.Error("expected fresh PoolEdge not to be stale")
	}
}

func TestIsStale_Old(t *testing.T) {
	p := PoolEdge{CalibratedAt: time.Now().Add(-2 * time.Minute)}
	if !p.IsStale(1 * time.Minute) {
		t.Error("expected old PoolEdge to be stale")
	}
}

// ---------------------------------------------------------------------------
// bestUniFeeTier
// ---------------------------------------------------------------------------

func TestBestUniFeeTier_NoPool(t *testing.T) {
	q := NewQuoterWithFunc(mockNoPool())
	marketp := newMarketParams(90)
	_, ok := q.bestUniFeeTier(nil, marketp, zeroAddr, oneEth, oraclePrice1e36, 10.0, 0)
	if ok {
		t.Error("expected no result with no pool")
	}
}

func TestBestUniFeeTier_AllAboveMaxSlippage(t *testing.T) {
	// tous les fee tiers retournent un slippage de 20% > maxSlippage 9.9%
	q := NewQuoterWithFunc(mockQuoteSingle(20.0, oneEth))
	marketp := newMarketParams(90)
	_, ok := q.bestUniFeeTier(nil, marketp, zeroAddr, oneEth, oraclePrice1e36, 9.9, 0)
	if ok {
		t.Error("expected no result when all slippages exceed max")
	}
}

func TestBestUniFeeTier_SelectsBestAmountOut(t *testing.T) {
	// fee=100 → amountOut petit, fee=500 → amountOut grand
	// les deux sous maxSlippage → doit choisir fee=500
	results := map[uint32]struct {
		slippage  float64
		amountOut *big.Int
	}{
		100:  {slippage: 0.5, amountOut: big.NewInt(900)},
		500:  {slippage: 0.3, amountOut: big.NewInt(950)},
		3000: {slippage: 0.8, amountOut: big.NewInt(800)},
	}
	q := NewQuoterWithFunc(mockByFee(results))
	marketp := newMarketParams(90)
	best, ok := q.bestUniFeeTier(nil, marketp, zeroAddr, big.NewInt(1000), oraclePrice1e36, 9.9, 0)
	if !ok {
		t.Fatal("expected a result")
	}
	if best.Fee != 500 {
		t.Errorf("expected fee=500 (best amountOut), got fee=%d", best.Fee)
	}
	if best.WCAmountOut.Cmp(big.NewInt(950)) != 0 {
		t.Errorf("expected amountOut=950, got %s", best.WCAmountOut)
	}
}

// ---------------------------------------------------------------------------
// Quote (div par 4)
// ---------------------------------------------------------------------------

func TestQuote_SuccessFirstTry(t *testing.T) {
	q := NewQuoterWithFunc(mockQuoteSingle(1.0, oneEth))
	marketp := newMarketParams(90) // maxSlippage = 9.9%
	result, ok := q.Quote(nil, marketp, zeroAddr, oneEth, oraclePrice1e36)
	if !ok {
		t.Fatal("expected a valid quote")
	}
	if result.WCSlippage != 1.0 {
		t.Errorf("expected slippage=1.0, got %f", result.WCSlippage)
	}
}

func TestQuote_NoPoolReturnsEmpty(t *testing.T) {
	q := NewQuoterWithFunc(mockNoPool())
	marketp := newMarketParams(90)
	_, ok := q.Quote(nil, marketp, zeroAddr, oneEth, oraclePrice1e36)
	if ok {
		t.Error("expected no result with no pool")
	}
}

func TestQuote_FallsBackToDividedAmount(t *testing.T) {
	calls := 0
	// échoue sur le premier amountIn, réussit sur amountIn/4
	fn := func(
		_ *connector.Connector,
		marketp morpho.MarketParams,
		addr common.Address,
		amountIn, oraclePrice *big.Int,
		fee uint32,
	) (PoolEdge, bool) {
		calls++
		threshold := new(big.Int).Div(oneEth, big.NewInt(4))
		if amountIn.Cmp(threshold) > 0 {
			return PoolEdge{}, false // trop grand
		}
		return PoolEdge{
			Fee:          fee,
			WCSlippage:   1.0,
			WCAmountIn:   new(big.Int).Set(amountIn),
			WCAmountOut:  new(big.Int).Set(amountIn),
			CalibratedAt: time.Now(),
		}, true
	}
	q := NewQuoterWithFunc(fn)
	marketp := newMarketParams(90)
	result, ok := q.Quote(nil, marketp, zeroAddr, oneEth, oraclePrice1e36)
	if !ok {
		t.Fatal("expected fallback to succeed")
	}
	// doit avoir essayé oneEth (4 fee tiers) puis oneEth/4
	if calls < 5 {
		t.Errorf("expected at least 5 calls (4 fails + 1 success), got %d", calls)
	}
	expected := new(big.Int).Div(oneEth, big.NewInt(4))
	if result.WCAmountIn.Cmp(expected) != 0 {
		t.Errorf("expected amountIn=%s, got %s", expected, result.WCAmountIn)
	}
}

// ---------------------------------------------------------------------------
// QuoteBinarySearch
// ---------------------------------------------------------------------------

func TestQuoteBinarySearch_FindsMax(t *testing.T) {
	// acceptable si amountIn <= oneEth/2
	threshold := new(big.Int).Div(oneEth, big.NewInt(2))

	fn := func(
		_ *connector.Connector,
		_ morpho.MarketParams,
		_ common.Address,
		amountIn, _ *big.Int,
		fee uint32,
	) (PoolEdge, bool) {
		if amountIn.Cmp(threshold) > 0 {
			return PoolEdge{}, false
		}
		return PoolEdge{
			Fee:          fee,
			WCSlippage:   1.0,
			WCAmountIn:   new(big.Int).Set(amountIn),
			WCAmountOut:  new(big.Int).Set(amountIn),
			CalibratedAt: time.Now(),
		}, true
	}

	q := NewQuoterWithFunc(fn)
	marketp := newMarketParams(90)
	result, ok := q.QuoteBinarySearch(nil, marketp, zeroAddr, oneEth, oraclePrice1e36)
	if !ok {
		t.Fatal("expected binary search to find a result")
	}
	// le résultat doit être proche du threshold
	if result.WCAmountIn.Cmp(threshold) > 0 {
		t.Errorf("amountIn %s exceeds threshold %s", result.WCAmountIn, threshold)
	}
}

func TestQuoteBinarySearch_NoPool(t *testing.T) {
	q := NewQuoterWithFunc(mockNoPool())
	marketp := newMarketParams(90)
	_, ok := q.QuoteBinarySearch(nil, marketp, zeroAddr, oneEth, oraclePrice1e36)
	if ok {
		t.Error("expected no result with no pool")
	}
}

func TestQuoteBinarySearch_AlwaysAcceptable(t *testing.T) {
	// tous les montants acceptables → doit retourner le plus grand (proche de amountIn)
	q := NewQuoterWithFunc(mockQuoteSingle(0.5, oneEth))
	marketp := newMarketParams(90)
	result, ok := q.QuoteBinarySearch(nil, marketp, zeroAddr, oneEth, oraclePrice1e36)
	if !ok {
		t.Fatal("expected a result")
	}
	// après 12 itérations de binary search, doit converger vers oneEth
	delta := new(big.Int).Sub(oneEth, result.WCAmountIn)
	maxDelta := new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil) // tolérance 1e12 wei
	if delta.Sign() < 0 || delta.Cmp(maxDelta) > 0 {
		t.Errorf("expected amountIn close to %s, got %s (delta=%s)", oneEth, result.WCAmountIn, delta)
	}
}
