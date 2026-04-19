package swap

import (
	"context"
	"math/big"

	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

// Struct qui correspond exactement au QuoteExactInputSingleParams du contrat
type QuoteExactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	AmountIn          *big.Int
	Fee               *big.Int // uint24
	SqrtPriceLimitX96 *big.Int // uint160
}

var (
	UniswapQuoterV2Addr = common.HexToAddress("0x3d4e44Eb1374240CE5F1B871ab261CD16335B76a")

	FuncQuoteExactInputSingleV2 = w3.MustNewFunc(
		`quoteExactInputSingle((address tokenIn, address tokenOut, uint256 amountIn, uint24 fee, uint160 sqrtPriceLimitX96) params)`,
		`uint256 amountOut, uint160 sqrtPriceX96After, uint32 initializedTicksCrossed, uint256 gasEstimate`,
	)
)

func Quote(client *w3.Client, marketp morpho.MarketParams, amountIn *big.Int) (*big.Int, error) {

	params := QuoteExactInputSingleParams{
		TokenIn:           marketp.CollateralToken,
		TokenOut:          marketp.LoanToken,
		AmountIn:          amountIn,
		Fee:               big.NewInt(int64(3000)),
		SqrtPriceLimitX96: big.NewInt(0),
	}

	var (
		amountOut               *big.Int
		sqrtPriceX96After       *big.Int
		initializedTicksCrossed uint32
		gasEstimate             *big.Int
		err                     error
	)

	if err := client.CallCtx(context.Background(),
		eth.CallFunc(UniswapQuoterV2Addr, FuncQuoteExactInputSingleV2, params).Returns(
			&amountOut,
			&sqrtPriceX96After,
			&initializedTicksCrossed,
			&gasEstimate,
		),
	); err != nil {
		return nil, err
	}

	return amountOut, err
}
