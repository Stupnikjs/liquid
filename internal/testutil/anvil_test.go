package testutil

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"testing"

	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("no .env file found, using system env")
	}
	fmt.Println(os.Getenv("BASE_HTTP_RPC_ALCH"))
	os.Exit(m.Run())

}

// WrapETH wrappe `amount` wei en WETH puis vérifie le solde.
func (a *AnvilInstance) WrapETH(t *testing.T, amount *big.Int) {
	txCtx, _ := a.newTxCtx(t, anvil_pk0)
	ctx := context.Background()
	receipt := sendAndWait(t, txCtx.client, ctx, txCtx.nonce, txCtx.gasPrice, txCtx.chainID, weth, amount, depositCalldata, txCtx.privKey)

	if receipt == nil {
		t.Fatalf("receipt non trouvé après timeout")
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("tx revertée, status: %d", receipt.Status)
	}

	// Vérifier le solde WETH via balanceOf(address)
	// ABI encode : sélecteur + adresse paddée à 32 bytes
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

func TestWrapETH(t *testing.T) {
	a := StartAnvilFork(t, os.Getenv("BASE_HTTP_RPC_ALCH"), 0)
	oneETH := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18))
	a.WrapETH(t, oneETH)
}

func (a *AnvilInstance) ApproveAndBorrow(t *testing.T, collateralAmount, borrowAmount *big.Int) {
	t.Helper()
	ctx := context.Background()
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

func TestBorrowUSDC(t *testing.T) {

	a := StartAnvilFork(t, os.Getenv("BASE_HTTP_RPC_ALCH"), 0)

	oneETH := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18))

	// D'abord wrap ETH → WETH
	a.WrapETH(t, oneETH)

	// Emprunter le max soit 1ETH en USDC * LLTV  (6 décimales)
	amount := new(big.Int).Mul(market.Lltv, big.NewInt(2300))
	borrowAmount := new(big.Int).Div(amount, big.NewInt(1e12))

	a.ApproveAndBorrow(t, oneETH, borrowAmount)

}

func (a *AnvilInstance) LiquidationSetup(t *testing.T, ethprice *big.Int) {
	oneETH := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18))
	// D'abord wrap ETH → WETH
	a.WrapETH(t, oneETH)
	// Emprunter le max soit 1ETH en USDC * LLTV  (6 décimales)
	// passer le prix de l'eth en usdc

	amount := new(big.Int).Mul(market.Lltv, ethprice)
	borrowAmount := new(big.Int).Div(amount, utils.TenPowInt(42))

	a.ApproveAndBorrow(t, oneETH, borrowAmount)
}
func TestBorrowUSDCAndLiquidate(t *testing.T) {

	a := StartAnvilFork(t, os.Getenv("BASE_HTTP_RPC_ALCH"), 0)
	ctx := context.Background()
	txCtx, err := a.newTxCtx(t, os.Getenv("BASE_PK")[2:])
	privKey, _ := crypto.HexToECDSA(os.Getenv("BASE_PK")[2:])
	sender := crypto.PubkeyToAddress(privKey.PublicKey)
	t.Logf("sender: %s", sender.Hex())

	ownerSel := crypto.Keccak256([]byte("owner()"))[:4]
	res, _ := txCtx.client.CallContract(ctx, ethereum.CallMsg{
		To:   &config.BaseLiquidatorUni,
		Data: ownerSel,
	}, nil)
	t.Logf("owner:  0x%x", res[12:32])
	t.Log(err)
	// lire le prix depuis l'oracle Morpho (retourne price avec 36 decimales)
	calldata := crypto.Keccak256([]byte("price()"))[:4]
	result, err := txCtx.client.CallContract(ctx, ethereum.CallMsg{
		To:   &market.Oracle,
		Data: calldata,
	}, nil)
	t.Log(err)
	price := new(big.Int).SetBytes(result)
	a.LiquidationSetup(t, price)

	// Après LiquidationSetup, avant d'envoyer la liquidation
	err = txCtx.client.Client().CallContext(ctx, nil, "evm_increaseTime", 240*60*24*180) // 6 mois en secondes
	if err != nil {
		t.Fatalf("evm_increaseTime: %v", err)
	}
	err = txCtx.client.Client().CallContext(ctx, nil, "evm_mine", nil)
	if err != nil {
		t.Fatalf("evm_mine: %v", err)
	}

	liqArg := lqtypes.LiquidateArgs{
		MarketParams: market,
		Borrower:     common.HexToAddress(FundedAccounts[0]),
		SeizedAssets: utils.WAD,
		RepaidShares: big.NewInt(0),
		SwapRouter:   config.BaseUniswapV3Router,
		PoolFee:      big.NewInt(100),
		MinOut:       big.NewInt(0),
	}

	calldata, err = lqtypes.EncodeLiquidateCalldata(liqArg)
	if err != nil {
		t.Fatalf("EncodeLiquidateCalldata: %v", err)
	}

	// time wrap or oracle manipulation

	receipt := sendAndWait(
		t,
		txCtx.client,
		context.Background(),
		txCtx.nonce,
		txCtx.gasPrice,
		txCtx.chainID,
		config.BaseLiquidatorUni, // to: le contrat liquidateur
		big.NewInt(0),            // value: 0 ETH
		calldata,                 // data: calldata encodé
		txCtx.privKey,
	)

	t.Logf("liquidation txHash: %s, status: %d", receipt.TxHash, receipt.Status)

}
