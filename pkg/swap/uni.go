package swap

import (
	"context"
	"math/big"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

var (
	uniQuoterV2 = common.HexToAddress("0x3d4e44Eb1374240CE5F1B136588e5fA50aA5e7C8") // Base
)

var feeTiers = []uint32{500, 3000, 10000} // 0.05%, 0.3%, 1%

type QuoteResult struct {
	AmountOut *big.Int
	Fee       uint32
}

// BestQuote essaie les 3 fee tiers et retourne le meilleur
func BestQuote(conn *connector.Connector, tokenIn, tokenOut common.Address, amountIn *big.Int) (*QuoteResult, error) {
	type quoteParams struct {
		TokenIn           common.Address
		TokenOut          common.Address
		AmountIn          *big.Int
		Fee               *big.Int
		SqrtPriceLimitX96 *big.Int
	}
	type QuoteOutput struct {
		AmountOut               *big.Int
		SqrtPriceX96After       *big.Int
		InitializedTicksCrossed *big.Int
		GasEstimate             *big.Int
	}

	outputs := make([]*QuoteOutput, len(feeTiers))
	callers := make([]w3types.RPCCaller, len(feeTiers))

	for i, fee := range feeTiers {
		outputs[i] = &QuoteOutput{}
		callers[i] = eth.CallFunc(uniQuoterV2, config.FuncQuoteExactInputSingle, quoteParams{
			TokenIn:           tokenIn,
			TokenOut:          tokenOut,
			AmountIn:          amountIn,
			Fee:               big.NewInt(int64(fee)),
			SqrtPriceLimitX96: big.NewInt(0),
		}).Returns(outputs[i])
	}

	if err := conn.EthCallCtx(context.Background(), callers); err != nil {
		return nil, err
	}

	best := &QuoteResult{AmountOut: big.NewInt(0)}
	for i, fee := range feeTiers {
		if outputs[i].AmountOut != nil && outputs[i].AmountOut.Cmp(best.AmountOut) > 0 {
			best.AmountOut = outputs[i].AmountOut
			best.Fee = fee
		}
	}
	return best, nil
}

// Calldata encode l'appel exactInputSingle pour le router
func Calldata(tokenIn, tokenOut common.Address, fee uint32, amountIn, minOut *big.Int, recipient common.Address) ([]byte, error) {
	type exactInputSingleParams struct {
		TokenIn           common.Address
		TokenOut          common.Address
		Fee               uint32
		Recipient         common.Address
		AmountIn          *big.Int
		AmountOutMinimum  *big.Int
		SqrtPriceLimitX96 *big.Int
	}

	return config.FuncQuoteExactInputSingle.EncodeArgs(exactInputSingleParams{
		TokenIn:           tokenIn,
		TokenOut:          tokenOut,
		Fee:               fee,
		Recipient:         recipient,
		AmountIn:          amountIn,
		AmountOutMinimum:  minOut,
		SqrtPriceLimitX96: big.NewInt(0),
	})
}
