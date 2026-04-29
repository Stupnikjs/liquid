package swap

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

// ---------------------------------------------------------------------------
// Adresses Base mainnet
// ---------------------------------------------------------------------------

// UniswapFees est la liste des fee tiers Uniswap V3 supportés (en pips).
var UniswapFees = []uint32{100, 500, 3000, 10000}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// QuoteExactInputSingleParams correspond au tuple attendu par le quoter Uniswap V3.
type QuoteExactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	AmountIn          *big.Int
	Fee               *big.Int
	SqrtPriceLimitX96 *big.Int
}

// QuoteResult contient le résultat d'un quote Uniswap V3.
type QuoteResult struct {
	AmountIn     *big.Int
	AmountOut    *big.Int
	NetAmountOut *big.Int // identique à AmountOut (fees déjà inclus dans le swap)
	Fee          uint32
	FeeAmount    *big.Int // estimation des fees en tokenIn
	Slippage     float64  // slippage par rapport au prix oracle, en %
}

// SingleQuoterFunc est la signature d'une fonction quotant un seul fee tier.
// Elle est injectable pour faciliter les tests (mocks, stubs).
type SingleQuoterFunc func(
	client *w3.Client,
	marketp morpho.MarketParams,
	uniswapQuoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
	fee uint32,
) (*QuoteResult, error)

// ---------------------------------------------------------------------------
// ABI Uniswap QuoterV2
// ---------------------------------------------------------------------------

var FuncQuoteExactInputSingleV2 = w3.MustNewFunc(
	`quoteExactInputSingle((address tokenIn, address tokenOut, uint256 amountIn, uint24 fee, uint160 sqrtPriceLimitX96) params)`,
	`uint256 amountOut, uint160 sqrtPriceX96After, uint32 initializedTicksCrossed, uint256 gasEstimate`,
)

// ---------------------------------------------------------------------------
// Quoter
// ---------------------------------------------------------------------------

// Quoter encapsule la logique de recherche du meilleur fee tier avec une
// dépendance injectable sur la fonction de quote unitaire.
type Quoter struct {
	quoteSingle SingleQuoterFunc
}

// NewQuoter retourne un Quoter utilisant les vrais appels RPC.
func NewQuoter() *Quoter {
	return &Quoter{quoteSingle: RPCQuoteSingle}
}

// NewQuoterWithFunc permet d'injecter un quoteSingle custom (tests, mocks).
func NewQuoterWithFunc(fn SingleQuoterFunc) *Quoter {
	return &Quoter{quoteSingle: fn}
}

// ---------------------------------------------------------------------------
// Fonctions package-level (raccourcis vers Quoter)
// ---------------------------------------------------------------------------

// Quote cherche le meilleur fee tier avec un slippage acceptable.
// Si le montant est trop grand, il est divisé par 4 successivement.
func Quote(client *w3.Client, marketp morpho.MarketParams, uniswapQuoterAddr common.Address, amountIn, oraclePrice *big.Int) (*QuoteResult, error) {
	return NewQuoter().Quote(client, marketp, uniswapQuoterAddr, amountIn, oraclePrice)
}

// QuoteBinarySearch trouve le montant maximal swappable avec un slippage acceptable.
func QuoteBinarySearch(client *w3.Client, marketp morpho.MarketParams, uniswapQuoterAddr common.Address, amountIn, oraclePrice *big.Int) (*QuoteResult, error) {
	return NewQuoter().QuoteBinarySearch(client, marketp, uniswapQuoterAddr, amountIn, oraclePrice)
}

// ---------------------------------------------------------------------------
// Méthodes Quoter
// ---------------------------------------------------------------------------

// Quote cherche le meilleur fee tier parmi UniswapFees avec un slippage ≤ MaxSlippage.
// Si aucun fee tier n'est acceptable pour amountIn, divise par 4 jusqu'à 1.
func (q *Quoter) Quote(
	client *w3.Client,
	marketp morpho.MarketParams,
	uniswapQuoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
) (*QuoteResult, error) {
	maxSlippage := MaxSlippage(marketp.LLTV)
	current := new(big.Int).Set(amountIn)

	for current.Sign() > 0 {
		best := q.bestFeeTier(client, marketp, uniswapQuoterAddr, current, oraclePrice, maxSlippage, 200*time.Millisecond)
		if best != nil {
			fmt.Printf("Pair %s/%s | source: uniswap | fee: %d | slippage: %.4f%%\n",
				marketp.CollateralTokenStr, marketp.LoanTokenStr, best.Fee, best.Slippage)
			return best, nil
		}
		current.Div(current, big.NewInt(4))
	}

	return nil, fmt.Errorf("no acceptable slippage found for %s -> %s",
		marketp.CollateralTokenStr, marketp.LoanTokenStr)
}

// QuoteBinarySearch trouve le montant maximal swappable avec slippage ≤ MaxSlippage
// en effectuant une recherche binaire (max 14 itérations).
func (q *Quoter) QuoteBinarySearch(
	client *w3.Client,
	marketp morpho.MarketParams,
	uniswapQuoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
) (*QuoteResult, error) {
	maxSlippage := MaxSlippage(marketp.LLTV)

	lo := big.NewInt(1)
	hi := new(big.Int).Set(amountIn)
	var best *QuoteResult

	for i := 0; i < 12 && lo.Cmp(hi) <= 0; i++ {
		mid := new(big.Int).Rsh(new(big.Int).Add(lo, hi), 1)

		result := q.bestFeeTier(client, marketp, uniswapQuoterAddr, mid, oraclePrice, maxSlippage, 300*time.Millisecond)
		if result != nil {
			best = result
			lo = new(big.Int).Add(mid, big.NewInt(1))
		} else {
			hi = new(big.Int).Sub(mid, big.NewInt(1))
		}
	}

	if best == nil {
		fmt.Println(fmt.Errorf("no acceptable slippage found for %s -> %s ",
			marketp.CollateralTokenStr, marketp.LoanTokenStr))
		return nil, fmt.Errorf("no acceptable slippage found for %s -> %s",
			marketp.CollateralTokenStr, marketp.LoanTokenStr)
	}
	fmt.Printf("acceptable slippage found for %s -> %s  %f \n",
		marketp.CollateralTokenStr, marketp.LoanTokenStr, best.Slippage)
	return best, nil
}

// bestFeeTier itère sur tous les fee tiers et retourne le meilleur résultat
// (amountOut maximal) avec un slippage ≤ maxSlippage, ou nil si aucun ne convient.
func (q *Quoter) bestFeeTier(
	client *w3.Client,
	marketp morpho.MarketParams,
	uniswapQuoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
	maxSlippage float64,
	rateLimit time.Duration,
) *QuoteResult {
	var best *QuoteResult
	for _, fee := range UniswapFees {
		time.Sleep(rateLimit)
		result, err := q.quoteSingle(client, marketp, uniswapQuoterAddr, amountIn, oraclePrice, fee)
		if err != nil || result == nil {
			continue
		}
		if result.Slippage <= maxSlippage {
			if best == nil || result.AmountOut.Cmp(best.AmountOut) > 0 {
				best = result
			}
		}
	}
	return best
}

// ---------------------------------------------------------------------------
// Implémentation RPC réelle
// ---------------------------------------------------------------------------

// RPCQuoteSingle appelle le contrat Uniswap QuoterV2 via RPC pour obtenir
// un quote exact input single pour un fee tier donné.
// Retourne nil, nil si le pool n'existe pas (erreur ignorée).
func RPCQuoteSingle(
	client *w3.Client,
	marketp morpho.MarketParams,
	uniswapQuoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
	fee uint32,
) (*QuoteResult, error) {
	params := QuoteExactInputSingleParams{
		TokenIn:           marketp.CollateralToken,
		TokenOut:          marketp.LoanToken,
		AmountIn:          amountIn,
		Fee:               big.NewInt(int64(fee)),
		SqrtPriceLimitX96: big.NewInt(0),
	}

	var (
		amountOut               *big.Int
		sqrtPriceX96After       *big.Int
		initializedTicksCrossed uint32
		gasEstimate             *big.Int
	)

	if err := client.CallCtx(context.Background(),
		eth.CallFunc(uniswapQuoterAddr, FuncQuoteExactInputSingleV2, params).Returns(
			&amountOut,
			&sqrtPriceX96After,
			&initializedTicksCrossed,
			&gasEstimate,
		),
	); err != nil {
		// Pool inexistante ou appel échoué : on skip silencieusement.
		return nil, nil
	}

	// Estimation des fees en tokenIn : amountIn * fee / 1_000_000
	feeAmount := new(big.Int).Div(
		new(big.Int).Mul(amountIn, big.NewInt(int64(fee))),
		big.NewInt(1_000_000),
	)

	return &QuoteResult{
		AmountIn:     amountIn,
		AmountOut:    amountOut,
		NetAmountOut: amountOut,
		Fee:          fee,
		FeeAmount:    feeAmount,
		Slippage:     ComputeSlippage(amountIn, amountOut, oraclePrice),
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers slippage
// ---------------------------------------------------------------------------

// MaxSlippage calcule le slippage maximal tolérable en % à partir du LLTV du marché.
// Formule : bonus de liquidation = 100 - LLTV%, moins 0.1% de marge pour le gas.
func MaxSlippage(lltv *big.Int) float64 {
	lltvF, _ := new(big.Float).SetInt(lltv).Float64()
	lltvPct := lltvF / 1e18 * 100
	bonus := 100 - lltvPct
	const gasCushion = 0.1
	return bonus - gasCushion
}

// ComputeSlippage calcule le slippage en % entre le montant attendu (selon l'oracle)
// et le montant réellement obtenu du swap.
//
// Formule : (expectedOut - amountOut) / expectedOut * 100
//
// oraclePrice doit être exprimé en 1e36 (convention Morpho).
// Retourne 0 si le prix oracle est nul ou si expectedOut est nul.
func ComputeSlippage(amountIn, amountOut *big.Int, oraclePrice *big.Int) float64 {
	if oraclePrice == nil || oraclePrice.Sign() == 0 {
		return 0
	}

	// expectedOut = amountIn * oraclePrice / 1e36
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	expectedOut := new(big.Int).Div(
		new(big.Int).Mul(amountIn, oraclePrice),
		scale,
	)

	if expectedOut.Sign() == 0 {
		return 0
	}

	diff := new(big.Int).Sub(expectedOut, amountOut)
	slip, _ := new(big.Float).Quo(
		new(big.Float).SetInt(diff),
		new(big.Float).SetInt(expectedOut),
	).Float64()

	return slip * 100
}
