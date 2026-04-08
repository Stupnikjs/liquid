package cex

import (
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
)

type Token string

func (t Token) String() string {
	return string(t)
}

type CexCache struct {
	BTC    atomic.Value
	ETH    atomic.Value
	XRP    atomic.Value
	EURUSD atomic.Value

	// volatilité par marché
	VolBTC MarketVolatility
	VolETH MarketVolatility
	VolXRP MarketVolatility
}

func NewCexCache() *CexCache {
	return &CexCache{
		BTC:    atomic.Value{},
		ETH:    atomic.Value{},
		XRP:    atomic.Value{},
		EURUSD: atomic.Value{},
	}
}

func (cex *CexCache) BTCtoBigInt() *big.Int {
	v := cex.BTC.Load()
	f, _ := v.(float64)
	return utils.FloatE36Int(f)
}

func (cex *CexCache) ETHtoBigInt() *big.Int {
	v := cex.ETH.Load()
	f, _ := v.(float64)
	return utils.FloatE36Int(f)
}

func (cex *CexCache) XRPtoBigInt() *big.Int {
	v := cex.XRP.Load()
	f, _ := v.(float64)
	return utils.FloatE36Int(f)
}

func (cex *CexCache) EURUSDtoBigInt() *big.Int {
	v := cex.EURUSD.Load()
	f, _ := v.(float64)
	return utils.FloatE36Int(f)
}

// Setters on-chain — appelés depuis PollOnChainRates
// Le rate vient du contrat en 1e18, on le convertit en float64 pour rester cohérent

func (cex *CexCache) UpdateNonCorrelated(priceUpdate PriceUpdate) {
	switch priceUpdate.ProductID {
	case "BTC-USD":
		cex.BTC.Store(priceUpdate.Price.MidPrice)
		cex.VolBTC.Push(priceUpdate.Price.MidPrice)
	case "ETH-USD":
		cex.ETH.Store(priceUpdate.Price.MidPrice)
		cex.VolETH.Push(priceUpdate.Price.MidPrice)
	case "XRP-USD":
		cex.XRP.Store(priceUpdate.Price.MidPrice)
		cex.VolXRP.Push(priceUpdate.Price.MidPrice)
	}
}

type MarketVolatility struct {
	prices [1000]float64 // ring buffer des 30 derniers prix
	idx    int           // position courante dans le buffer
	count  int           // nb de prix enregistrés (max 30)
	mu     sync.RWMutex
}

func (mv *MarketVolatility) Push(price float64) {
	mv.mu.Lock()
	defer mv.mu.Unlock()
	mv.prices[mv.idx] = price
	mv.idx = (mv.idx + 1) % len(mv.prices)
	if mv.count < len(mv.prices) {
		mv.count++
	}
}

func (mv *MarketVolatility) DownsideStdDev() float64 {
	mv.mu.RLock()
	defer mv.mu.RUnlock()
	if mv.count < 2 {
		return 0
	}
	var sum float64
	for i := 0; i < mv.count; i++ {
		sum += mv.prices[i]
	}
	mean := sum / float64(mv.count)
	if mean == 0 {
		return 0
	}
	var variance float64
	var n int
	for i := 0; i < mv.count; i++ {
		d := mv.prices[i] - mean
		if d < 0 { // seulement les déviations négatives
			variance += d * d
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return math.Sqrt(variance/float64(n)) * 10_000 / mean
}

func (cex *CexCache) GetRefreshParams(treshold float64) (time.Duration, float64) {
	// plus c'est volatile, plus on refresh souvent
	maxVol := math.Max(math.Max(cex.VolBTC.DownsideStdDev(), cex.VolETH.DownsideStdDev()), cex.VolXRP.DownsideStdDev())
	var interval time.Duration
	switch {
	case maxVol > 20:
		interval = 2 * time.Second
	case maxVol > 10:
		interval = 20 * time.Second
	default:
		interval = 60 * time.Second
	}

	return interval, maxVol
}

func GetCollateralPriceInLoan(cex *CexCache, m *morpho.MarketParams) *big.Int {
	switch {
	case m.CollateralTokenStr == "cbBTC" && m.LoanTokenStr == "USDC":
		return cex.BTCtoBigInt()

	case m.CollateralTokenStr == "WETH" && m.LoanTokenStr == "USDC":
		return cex.ETHtoBigInt()

	case m.CollateralTokenStr == "cbXRP" && m.LoanTokenStr == "USDC":
		return cex.XRPtoBigInt()

	case m.CollateralTokenStr == "cbETH" && m.LoanTokenStr == "USDC":
		return cex.ETHtoBigInt()

	default:
		return nil
	}
}
