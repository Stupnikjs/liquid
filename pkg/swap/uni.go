package swap

import (
	"context"
	"fmt"
	"math/big"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

var (
	// Uniswap V3
	FuncQuoteExactInputSingle = w3.MustNewFunc(
		`quoteExactInputSingle((address tokenIn, address tokenOut, uint256 amountIn, uint24 fee, uint160 sqrtPriceLimitX96))`,
		`uint256 amountOut, uint160 sqrtPriceX96After, uint32 initializedTicksCrossed, uint256 gasEstimate`,
	)
	FuncQuoteExactInput = w3.MustNewFunc(
		`quoteExactInput(bytes path, uint256 amountIn)`,
		`uint256 amountOut, uint160[] sqrtPriceX96AfterList, uint32[] initializedTicksCrossedList, uint256 gasEstimate`,
	)

	// Aerodrome CL
	FuncAeroCLQuote = w3.MustNewFunc(
		`quoteExactInputSingle((address tokenIn, address tokenOut, uint256 amountIn, int24 tickSpacing, uint160 sqrtPriceLimitX96))`,
		`uint256 amountOut, uint160 sqrtPriceX96After, uint32 initializedTicksCrossed, uint256 gasEstimate`,
	)

	// Aerodrome V2
	FuncAeroV2GetAmountsOut = w3.MustNewFunc(
		`getAmountsOut(uint256 amountIn, (address from, address to, bool stable, address factory)[] routes)`,
		`uint256[] amounts`,
	)

	// addresses
	UniswapQuoterV2Addr    = common.HexToAddress("0xC5290058841028F1614F3A6F0F5816cAd0df5E27")
	AerodromeCLQuoterAddr  = common.HexToAddress("0x254cF9E1E6e233aa1AC962CB9B05b2cfeAaE15b0")
	AerodromeV2RouterAddr  = common.HexToAddress("0xcF77a3Ba9A5CA399B7c97c74d54e5b1Beb874E43")
	AerodromeV2FactoryAddr = common.HexToAddress("0x420DD381b31aEf6683db6B902084cB0FFECe40Da")

	// tokens intermédiaires pour multi-hop
	WETH = common.HexToAddress("0x4200000000000000000000000000000000000006")
	USDC = common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")
)

// ---- types ----

type QuoteResult struct {
	AmountOut *big.Int
	Fee       uint32
	Stable    bool
	Source    string
}

type Quoter interface {
	Quote(client *w3.Client, marketp morpho.MarketParams, amountIn *big.Int) (*QuoteResult, error)
}

// ---- Uniswap single hop ----

type UniswapQuoter struct {
	Fee uint32
}

func (u UniswapQuoter) Quote(client *w3.Client, marketp morpho.MarketParams, amountIn *big.Int) (*QuoteResult, error) {
	type QuoteParams struct {
		TokenIn           common.Address
		TokenOut          common.Address
		AmountIn          *big.Int
		Fee               *big.Int
		SqrtPriceLimitX96 *big.Int
	}

	params := QuoteParams{
		TokenIn:           marketp.CollateralToken,
		TokenOut:          marketp.LoanToken,
		AmountIn:          amountIn,
		Fee:               big.NewInt(int64(u.Fee)),
		SqrtPriceLimitX96: big.NewInt(0),
	}

	var (
		amountOut               *big.Int
		sqrtPriceX96After       *big.Int
		initializedTicksCrossed uint32
		gasEstimate             *big.Int
	)

	if err := client.CallCtx(context.Background(),
		eth.CallFunc(UniswapQuoterV2Addr, FuncQuoteExactInputSingle, params).Returns(
			&amountOut,
			&sqrtPriceX96After,
			&initializedTicksCrossed,
			&gasEstimate,
		),
	); err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &QuoteResult{AmountOut: amountOut, Fee: u.Fee, Source: "uniswap"}, nil
}

// ---- Aerodrome CL ----

type AerodromeCLQuoter struct {
	TickSpacing *big.Int
}

func (a AerodromeCLQuoter) Quote(client *w3.Client, marketp morpho.MarketParams, amountIn *big.Int) (*QuoteResult, error) {
	type AeroCLParams struct {
		TokenIn           common.Address
		TokenOut          common.Address
		AmountIn          *big.Int
		TickSpacing       *big.Int
		SqrtPriceLimitX96 *big.Int
	}

	params := AeroCLParams{
		TokenIn:           marketp.CollateralToken,
		TokenOut:          marketp.LoanToken,
		AmountIn:          amountIn,
		TickSpacing:       a.TickSpacing,
		SqrtPriceLimitX96: big.NewInt(0),
	}

	var amountOut *big.Int
	if err := client.CallCtx(context.Background(),
		eth.CallFunc(AerodromeCLQuoterAddr, FuncAeroCLQuote, params).Returns(&amountOut),
	); err != nil {
		return nil, err
	}

	return &QuoteResult{
		AmountOut: amountOut,
		Source:    fmt.Sprintf("aerodrome-cl-tick%s", a.TickSpacing.String()),
	}, nil
}

// ---- Aerodrome V2 ----

type AerodromeV2Quoter struct {
	Stable bool
}

type aeroRoute struct {
	From    common.Address
	To      common.Address
	Stable  bool
	Factory common.Address
}

func (a AerodromeV2Quoter) Quote(client *w3.Client, marketp morpho.MarketParams, amountIn *big.Int) (*QuoteResult, error) {
	routes := []aeroRoute{{
		From:    marketp.CollateralToken,
		To:      marketp.LoanToken,
		Stable:  a.Stable,
		Factory: AerodromeV2FactoryAddr,
	}}

	var amountsOut []*big.Int
	if err := client.CallCtx(context.Background(),
		eth.CallFunc(AerodromeV2RouterAddr, FuncAeroV2GetAmountsOut, amountIn, routes).Returns(&amountsOut),
	); err != nil {
		return nil, err
	}

	if len(amountsOut) < 2 || amountsOut[1] == nil || amountsOut[1].Sign() == 0 {
		return nil, fmt.Errorf("aerodrome v2: invalid response")
	}

	source := "aerodrome-v2-volatile"
	if a.Stable {
		source = "aerodrome-v2-stable"
	}

	return &QuoteResult{AmountOut: amountsOut[1], Stable: a.Stable, Source: source}, nil
}

// ---- FindBestPool ----

func FindBestPool(client *w3.Client, marketp morpho.MarketParams, amountIn *big.Int, oraclePrice *big.Int) (*big.Int, *QuoteResult, float64) {

	if amountIn == nil || amountIn.Sign() == 0 {
		return nil, nil, 100.0
	}
	if oraclePrice == nil || oraclePrice.Sign() == 0 {
		return nil, nil, 100.0
	}

	pow36 := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	expectedOut := new(big.Int).Mul(amountIn, oraclePrice)
	expectedOut.Div(expectedOut, pow36)

	if expectedOut.Sign() == 0 {
		fmt.Println("expectedOut is zero")
		return nil, nil, 100.0
	}

	maxSlippage, _ := new(big.Int).Sub(utils.WAD, marketp.LLTV).Float64()
	maxSlippage /= 2e16

	quoters := []Quoter{
		// single hop ETH-corrélé
		AerodromeCLQuoter{TickSpacing: big.NewInt(1)},
		UniswapQuoter{Fee: 100},
		UniswapQuoter{Fee: 500},
		UniswapQuoter{Fee: 3000},
		UniswapQuoter{Fee: 10000},
		// Aerodrome CL
		AerodromeCLQuoter{TickSpacing: big.NewInt(50)},
		AerodromeCLQuoter{TickSpacing: big.NewInt(100)},
		AerodromeCLQuoter{TickSpacing: big.NewInt(200)},
		AerodromeCLQuoter{TickSpacing: big.NewInt(2000)},
		// Aerodrome V2
		AerodromeV2Quoter{Stable: false},
		AerodromeV2Quoter{Stable: true},
		// multi-hop via WETH

	}
	bestSlippage := 100.0
	var bestResult *QuoteResult
	var bestAmount *big.Int
	found := false

	divisors := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}

	for _, divisor := range divisors {
		currentAmount := new(big.Int).Div(amountIn, divisor)
		if currentAmount.Sign() == 0 {
			continue
		}

		for _, q := range quoters {
			result, err := q.Quote(client, marketp, currentAmount)
			if err != nil || result == nil || result.AmountOut == nil || result.AmountOut.Sign() == 0 {
				continue
			}

			diff := new(big.Int).Sub(expectedOut, result.AmountOut)
			slippagePct := new(big.Float).Quo(
				new(big.Float).SetInt(diff),
				new(big.Float).SetInt(expectedOut),
			)
			slippagePct.Mul(slippagePct, big.NewFloat(100))
			slip, _ := slippagePct.Float64()

			if !found || slip < bestSlippage {
				bestSlippage = slip
				bestResult = result
				bestAmount = new(big.Int).Set(currentAmount)
				found = true
			}
		}

		if found && bestSlippage < maxSlippage {
			return bestAmount, bestResult, bestSlippage
		}
	}

	if !found {
		return nil, nil, 100.0
	}

	return bestAmount, bestResult, bestSlippage
}
