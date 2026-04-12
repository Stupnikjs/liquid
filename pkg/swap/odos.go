package swap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/lmittmann/w3"
)

type OdosQuoteRequest struct {
	InputTokens          []OdosToken `json:"inputTokens"`
	OutputTokens         []OdosToken `json:"outputTokens"`
	SlippageLimitPercent float64     `json:"slippageLimitPercent"`
	ChainId              int         `json:"chainId"`
}

type OdosToken struct {
	TokenAddress string `json:"tokenAddress"`
	Amount       string `json:"amount"`
}

type OdosQuoteResponse struct {
	OutAmounts  []string `json:"outAmounts"`
	PriceImpact float64  `json:"priceImpact"`
}

type OdosAssembleRequest struct {
	PathId   string `json:"pathId"`
	UserAddr string `json:"userAddr"`
}

type OdosAssembleResponse struct {
	Transaction struct {
		Data string `json:"data"`
	} `json:"transaction"`
}

func FindBestPool(client *w3.Client, tokenIn, tokenOut common.Address, amountIn *big.Int, oraclePrice *big.Int) (float64, float64) {

	// expectedOut = amountIn * oraclePrice / 1e36
	var expectedOut *big.Int
	expectedOut = new(big.Int).Mul(amountIn, oraclePrice)
	expectedOut.Div(expectedOut, utils.TenPowInt(36))
	amountOut, priceImpact, err := QuoteSwapOdos(tokenIn, tokenOut, amountIn)
	if err != nil {
		return 0, 100.0
	}

	diff := new(big.Int).Sub(expectedOut, amountOut)
	slippagePct := new(big.Float).Quo(
		new(big.Float).SetInt(diff),
		new(big.Float).SetInt(expectedOut),
	)
	slippagePct.Mul(slippagePct, big.NewFloat(100))
	slip, _ := slippagePct.Float64()

	return priceImpact, slip
}

func QuoteSwapOdos(tokenIn, tokenOut common.Address, amountIn *big.Int) (*big.Int, float64, error) {
	reqBody := OdosQuoteRequest{
		InputTokens: []OdosToken{{
			TokenAddress: tokenIn.Hex(),
			Amount:       amountIn.String(),
		}},
		OutputTokens: []OdosToken{{
			TokenAddress: tokenOut.Hex(),
			Amount:       "1", // proportion
		}},
		SlippageLimitPercent: 1.0,
		ChainId:              8453, // Base
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, 0, err
	}

	resp, err := http.Post("https://api.odos.xyz/sor/quote/v2", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	var result OdosQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, err
	}

	if len(result.OutAmounts) == 0 {
		return nil, 0, fmt.Errorf("no output amounts from Odos")
	}

	amountOut, ok := new(big.Int).SetString(result.OutAmounts[0], 10)
	if !ok {
		return nil, 0, fmt.Errorf("failed to parse amountOut: %s", result.OutAmounts[0])
	}

	return amountOut, result.PriceImpact, nil
}

func AssembleOdos(pathId string, liquidatorAddr common.Address) ([]byte, error) {
	reqBody := OdosAssembleRequest{
		PathId:   pathId,
		UserAddr: liquidatorAddr.Hex(),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post("https://api.odos.xyz/sor/assemble", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("odos assemble error %d: %s", resp.StatusCode, string(b))
	}

	var result OdosAssembleResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Transaction.Data == "" {
		return nil, fmt.Errorf("empty calldata from Odos assemble")
	}

	return hexutil.Decode(result.Transaction.Data)
}
