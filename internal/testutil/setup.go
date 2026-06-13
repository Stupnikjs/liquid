package testutil

import (
	"context"
	"math/big"
	"testing"

	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/Stupnikjs/liquid/pkg/swap"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func encodeMarketParams(m morpho.MarketParams) []byte {
	buf := make([]byte, 5*32)
	copy(buf[12:32], m.LoanToken.Bytes())
	copy(buf[32+12:64], m.CollateralToken.Bytes())
	copy(buf[64+12:96], m.Oracle.Bytes())
	copy(buf[96+12:128], m.Irm.Bytes())
	m.LLTV.FillBytes(buf[128:160])
	return buf
}

func (a *AnvilInstance) SetOraclePrice(t *testing.T, client *ethclient.Client, oracle common.Address, price *big.Int) {
	priceBytes := make([]byte, 32)
	price.FillBytes(priceBytes)

	code := []byte{0x7f}
	code = append(code, priceBytes...)
	code = append(code, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3)

	err := client.Client().Call(nil, "anvil_setCode", oracle.Hex(), hexutil.Encode(code))
	if err != nil {
		t.Fatalf("anvil_setCode: %v", err)
	}
}

func buildWETHUSDCSwapStep() swap.SwapStep {
	exactSingleInputMethod := swap.UniExactInputSingleMethod()

	type ExactInputSingleParams struct {
		TokenIn           common.Address
		TokenOut          common.Address
		Fee               *big.Int
		Recipient         common.Address
		AmountIn          *big.Int
		AmountOutMinimum  *big.Int
		SqrtPriceLimitX96 *big.Int
	}

	data, _ := exactSingleInputMethod.Inputs.Pack(ExactInputSingleParams{
		TokenIn:           weth,
		TokenOut:          usdc,
		Fee:               big.NewInt(500),
		Recipient:         config.BaseLiquidatorLast,
		AmountIn:          big.NewInt(0),
		AmountOutMinimum:  big.NewInt(0),
		SqrtPriceLimitX96: big.NewInt(0),
	})

	// ajouter le selector manuellement
	selector := exactSingleInputMethod.ID // 4 bytes
	calldata := append(selector, data...)

	return swap.SwapStep{
		Target:         config.BaseUniswapV3Router,
		Data:           calldata,
		TokenIn:        weth,
		TokenOut:       usdc,
		AmountInOffset: big.NewInt(132),
	}
}

// WrapETH wrappe `amount` wei en WETH puis vérifie le solde.
func (a *AnvilInstance) WrapETH(t *testing.T, privekeyhex string, amount *big.Int) {
	txCtx, _ := a.newTxCtx(t, privekeyhex)
	ctx := context.Background()
	receipt := sendAndWait(t, txCtx.client, ctx, txCtx.nonce, txCtx.gasPrice, txCtx.chainID, weth, amount, depositCalldata, txCtx.privKey)

	if receipt == nil {
		t.Fatalf("receipt non trouvé après timeout")
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("tx revertée, status: %d", receipt.Status)
	}

	calldata := make([]byte, 4+32)
	copy(calldata[:4], balanceOfSelector)
	copy(calldata[4+12:], me.Bytes()) // adresse = 12 bytes de padding + 20 bytes

	result, err := txCtx.client.CallContract(ctx, ethereum.CallMsg{
		To:   &weth,
		Data: calldata,
	}, nil)
	if err != nil {
		t.Fatalf("balanceOf call: %v", err)
	}

	balance := new(big.Int).SetBytes(result)

	t.Logf("✓ WETH balance: %s wei", balance)

	if balance.Cmp(amount) != 0 {
		t.Fatalf("balance WETH attendu %s, obtenu %s", amount, balance)
	}
	if balance.Cmp(amount) == 0 {
		t.Logf("✓ WETH balance: %s wei", balance)
	}

}

func (a *AnvilInstance) ApproveAndBorrow(t *testing.T, collateralAmount, borrowAmount *big.Int) {
	t.Helper()
	ctx := context.Background()
	// borrowing as anvil_pk0
	txCtx, _ := a.newTxCtx(t, anvil_pk0)

	// 1. Approve WETH → Morpho
	{
		// approve(address spender, uint256 amount)
		data := make([]byte, 4+32+32)
		copy(data[:4], approveSelector)
		copy(data[4+12:36], morphoBlue.Bytes())
		collateralAmount.FillBytes(data[36:68])

		t.Log("1. Approve WETH → Morpho")
		sendAndWait(t, txCtx.client, ctx, txCtx.nonce, txCtx.gasPrice, txCtx.chainID, weth, big.NewInt(0), data, txCtx.privKey)
		txCtx.nonce++
	}

	// 2. SupplyCollateral
	{
		// supplyCollateral(MarketParams, uint256 assets, address onBehalf, bytes calldata data)
		marketEncoded := encodeMarketParams(market)

		// layout : selector | marketParams (160) | assets (32) | onBehalf offset (32) | data offset (32) | onBehalf (32) | data length (32)
		data := make([]byte, 4+160+32+32+32+32)
		copy(data[:4], supplyCollateralSelector)
		copy(data[4:164], marketEncoded)
		collateralAmount.FillBytes(data[164:196])
		copy(data[196+12:228], me.Bytes()) // onBehalf
		big.NewInt(256).FillBytes(data[228:260])
		// length = 0, data[260:292] déjà à zéro
		sig := crypto.Keccak256([]byte("supplyCollateral((address,address,address,address,uint256),uint256,address,bytes)"))
		t.Logf("supplyCollateral selector: %x", sig[:4])
		t.Log("2. SupplyCollateral WETH → Morpho")

		sendAndWait(t, txCtx.client, ctx, txCtx.nonce, txCtx.gasPrice, txCtx.chainID, morphoBlue, big.NewInt(0), data, txCtx.privKey)
		txCtx.nonce++
	}

	// 3. Borrow USDC
	{
		// borrow(MarketParams, uint256 assets, uint256 shares, address receiver, address onBehalf)
		marketEncoded := encodeMarketParams(market)
		data := make([]byte, 4+160+32+32+32+32)
		copy(data[:4], borrowSelector)
		copy(data[4:164], marketEncoded)
		borrowAmount.FillBytes(data[164:196]) // assets
		// shares = 0 → déjà zéro
		copy(data[228+12:260], me.Bytes()) // receiver
		copy(data[260+12:292], me.Bytes()) // onBehalf

		t.Log("3. Borrow USDC depuis Morpho")
		sendAndWait(t, txCtx.client, ctx, txCtx.nonce, txCtx.gasPrice, txCtx.chainID, morphoBlue, big.NewInt(0), data, txCtx.privKey)
		txCtx.nonce++
	}

	// Vérifier le solde USDC reçu
	{
		calldata := make([]byte, 4+32)
		copy(calldata[:4], balanceOfSelector)
		copy(calldata[4+12:], me.Bytes())

		result, err := txCtx.client.CallContract(ctx, ethereum.CallMsg{
			To:   &usdc,
			Data: calldata,
		}, nil)
		if err != nil {
			t.Fatalf("balanceOf USDC: %v", err)
		}

		balance := new(big.Int).SetBytes(result)
		t.Logf("✓ USDC balance: %s (unités USDC × 1e6)", balance)

		if balance.Cmp(big.NewInt(0)) == 0 {
			t.Fatal("balance USDC toujours à 0 après borrow")
		}
	}
}

func (a *AnvilInstance) LiquidationSetup(t *testing.T, ethprice *big.Int) {
	oneETH := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18))
	// D'abord wrap ETH → WETH
	a.WrapETH(t, anvil_pk0, oneETH)
	// Emprunter le max soit 1ETH en USDC * LLTV  (6 décimales)
	// passer le prix de l'eth en usdc

	amount := new(big.Int).Mul(market.LLTV, ethprice)
	amount.Mul(amount, big.NewInt(95)) // 95% du LLTV max
	amount.Div(amount, big.NewInt(100))
	borrowAmount := new(big.Int).Div(amount, utils.TenPowInt(36))

	t.Logf("ethprice: %s", ethprice.String())
	t.Logf("borrowAmount: %s", borrowAmount.String())
	a.ApproveAndBorrow(t, oneETH, borrowAmount)
}
