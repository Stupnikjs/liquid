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

type QuoteExactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	AmountIn          *big.Int
	Fee               *big.Int
	SqrtPriceLimitX96 *big.Int
}

type QuoteResult struct {
	AmountIn     *big.Int
	AmountOut    *big.Int
	NetAmountOut *big.Int // après fees
	Fee          uint32
	FeeAmount    *big.Int // montant des fees en tokenIn
	Slippage     float64  // en %
}

var (
	FuncQuoteExactInputSingleV2 = w3.MustNewFunc(
		`quoteExactInputSingle((address tokenIn, address tokenOut, uint256 amountIn, uint24 fee, uint160 sqrtPriceLimitX96) params)`,
		`uint256 amountOut, uint160 sqrtPriceX96After, uint32 initializedTicksCrossed, uint256 gasEstimate`,
	)
)

var UniswapFees = []uint32{100, 500, 3000, 10000}

// SingleQuoterFunc est la signature d'une fonction qui quote un seul fee tier.
// Permet de mocker les appels RPC dans les tests.
type SingleQuoterFunc func(
	client *w3.Client,
	marketp morpho.MarketParams,
	uniswapQuoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
	fee uint32,
) (*QuoteResult, error)

// Quoter regroupe la logique de quote avec une dépendance injectable.
type Quoter struct {
	quoteSingle SingleQuoterFunc
}

// NewQuoter retourne un Quoter utilisant l'implémentation RPC réelle.
func NewQuoter() *Quoter {
	return &Quoter{quoteSingle: rpcQuoteSingle}
}

// NewQuoterWithFunc permet d'injecter un quoteSingle custom (tests, mocks).
func NewQuoterWithFunc(fn SingleQuoterFunc) *Quoter {
	return &Quoter{quoteSingle: fn}
}

func MaxSlippage(lltv *big.Int) float64 {
	lltvF, _ := new(big.Float).SetInt(lltv).Float64()
	lltvPct := lltvF / 1e18 * 100
	bonus := 100 - lltvPct
	gas := 0.1
	return bonus - gas
}

// Quote reste utilisable directement (utilise l'implémentation RPC réelle).
func Quote(client *w3.Client, marketp morpho.MarketParams, uniswapQuoterAddr common.Address, amountIn, oraclePrice *big.Int) (*QuoteResult, error) {
	return NewQuoter().Quote(client, marketp, uniswapQuoterAddr, amountIn, oraclePrice)
}

// QuoteBinarySearch reste utilisable directement.
func QuoteBinarySearch(client *w3.Client, marketp morpho.MarketParams, uniswapQuoterAddr common.Address, amountIn, oraclePrice *big.Int) (*QuoteResult, error) {
	return NewQuoter().QuoteBinarySearch(client, marketp, uniswapQuoterAddr, amountIn, oraclePrice)
}

// Quote cherche le meilleur fee tier avec slippage acceptable,
// en divisant par 4 si le montant est trop grand.
func (q *Quoter) Quote(client *w3.Client, marketp morpho.MarketParams, uniswapQuoterAddr common.Address, amountIn, oraclePrice *big.Int) (*QuoteResult, error) {
	var best *QuoteResult
	current := new(big.Int).Set(amountIn)
	maxSlippage := MaxSlippage(marketp.LLTV)

	for current.Sign() > 0 {
		for _, fee := range UniswapFees {
			time.Sleep(200 * time.Millisecond)
			result, err := q.quoteSingle(client, marketp, uniswapQuoterAddr, current, oraclePrice, fee)
			if err != nil || result == nil {
				continue
			}
			if result.Slippage <= maxSlippage {
				fmt.Printf("Pair %s/%s | source: uniswap | slippage: %f%%\n",
					marketp.CollateralTokenStr, marketp.LoanTokenStr, result.Slippage)
				if best == nil || result.AmountOut.Cmp(best.AmountOut) > 0 {
					best = result
				}
			}
		}

		if best != nil {
			return best, nil
		}

		current.Div(current, big.NewInt(4))
	}

	return nil, fmt.Errorf("no acceptable slippage found for %s -> %s",
		marketp.CollateralTokenStr, marketp.LoanTokenStr)
}

// QuoteBinarySearch trouve le montant maximal swappable avec slippage acceptable.
func (q *Quoter) QuoteBinarySearch(client *w3.Client, marketp morpho.MarketParams, uniswapQuoterAddr common.Address, amountIn, oraclePrice *big.Int) (*QuoteResult, error) {
	maxSlippage := MaxSlippage(marketp.LLTV)

	tryAmount := func(amount *big.Int) (*QuoteResult, error) {
		var best *QuoteResult
		for _, fee := range UniswapFees {
			time.Sleep(400 * time.Millisecond)
			result, err := q.quoteSingle(client, marketp, uniswapQuoterAddr, amount, oraclePrice, fee)
			if err != nil || result == nil {
				continue
			}
			if result.Slippage <= maxSlippage {
				if best == nil || result.AmountOut.Cmp(best.AmountOut) > 0 {
					best = result
				}
			}
		}
		return best, nil
	}

	lo := big.NewInt(1)
	hi := new(big.Int).Set(amountIn)
	var best *QuoteResult

	for i := 0; i < 14 && lo.Cmp(hi) <= 0; i++ {
		mid := new(big.Int).Add(lo, hi)
		mid.Rsh(mid, 1)

		result, err := tryAmount(mid)
		if err != nil {
			return nil, err
		}

		if result != nil {
			best = result
			lo = new(big.Int).Add(mid, big.NewInt(1))

		} else {
			hi = new(big.Int).Sub(mid, big.NewInt(1))

		}
	}

	if best == nil {
		return nil, fmt.Errorf("no acceptable slippage found for %s -> %s",
			marketp.CollateralTokenStr, marketp.LoanTokenStr)
	}
	return best, nil
}

// rpcQuoteSingle est l'implémentation réelle qui appelle le contrat Uniswap.
func rpcQuoteSingle(client *w3.Client, marketp morpho.MarketParams, uniswapQuoterAddr common.Address, amountIn, oraclePrice *big.Int, fee uint32) (*QuoteResult, error) {
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
		return nil, nil // pool inexistante, on skip
	}

	feeAmount := new(big.Int).Mul(amountIn, big.NewInt(int64(fee)))
	feeAmount.Div(feeAmount, big.NewInt(1_000_000))

	slippage := computeSlippage(amountIn, amountOut, oraclePrice)

	return &QuoteResult{
		AmountIn:     amountIn,
		AmountOut:    amountOut,
		NetAmountOut: amountOut,
		Fee:          fee,
		FeeAmount:    feeAmount,
		Slippage:     slippage,
	}, nil
}

func computeSlippage(amountIn, amountOut *big.Int, oraclePrice *big.Int) float64 {
	if oraclePrice == nil || oraclePrice.Sign() == 0 {
		return 0
	}

	expectedOut := new(big.Int).Mul(amountIn, oraclePrice)
	expectedOut.Div(expectedOut, new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil))

	if expectedOut.Sign() == 0 {
		return 0
	}

	diff := new(big.Int).Sub(expectedOut, amountOut)
	diffF := new(big.Float).SetInt(diff)
	expectedF := new(big.Float).SetInt(expectedOut)
	slippageF := new(big.Float).Quo(diffF, expectedF)
	slip, _ := slippageF.Float64()

	return slip * 100
}
