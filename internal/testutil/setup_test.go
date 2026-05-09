package testutil

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/Stupnikjs/liquid/internal/liquidate"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/config"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
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
	t.Helper()
	ctx := context.Background()

	// Dial go-ethereum natif
	client, err := ethclient.Dial(a.RPCURL)
	if err != nil {
		t.Fatalf("ethclient dial: %v", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("chainID: %v", err)
	}

	nonce, err := client.PendingNonceAt(ctx, me)
	if err != nil {
		t.Fatalf("nonce: %v", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		t.Fatalf("gasPrice: %v", err)
	}

	// Estimation du gas pour deposit()
	gas, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From:  me,
		To:    &weth,
		Value: amount,
		Data:  depositCalldata,
	})
	if err != nil {
		t.Fatalf("estimateGas: %v", err)
	}

	// Construire la tx legacy (Anvil accepte les deux)
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &weth,
		Value:    amount,
		Gas:      gas,
		GasPrice: gasPrice,
		Data:     depositCalldata,
	})

	// Signer avec la clé privée du compte 0 d'Anvil (déterministe)
	// Clé privée d'Anvil account[0] : ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
	privKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		t.Fatalf("privKey: %v", err)
	}

	signer := types.LatestSignerForChainID(chainID)
	signed, err := types.SignTx(tx, signer, privKey)
	if err != nil {
		t.Fatalf("signTx: %v", err)
	}

	if err := client.SendTransaction(ctx, signed); err != nil {
		t.Fatalf("sendTransaction: %v", err)
	}

	t.Logf("deposit txHash: %s", signed.Hash())

	// Attendre le minage (Anvil mine toutes les secondes)
	// une func dediée
	var receipt *types.Receipt
	for i := 0; i < 20; i++ {
		time.Sleep(300 * time.Millisecond)
		receipt, err = client.TransactionReceipt(ctx, signed.Hash())
		if err == nil {
			break
		}
	}
	if receipt == nil {
		t.Fatalf("receipt non trouvé après timeout")
	}

	t.Log(receipt)
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("tx revertée, status: %d", receipt.Status)
	}

	// Vérifier le solde WETH via balanceOf(address)
	// ABI encode : sélecteur + adresse paddée à 32 bytes
	calldata := make([]byte, 4+32)
	copy(calldata[:4], balanceOfSelector)
	copy(calldata[4+12:], me.Bytes()) // adresse = 12 bytes de padding + 20 bytes

	result, err := client.CallContract(ctx, ethereum.CallMsg{
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

	client, err := ethclient.Dial(a.RPCURL)
	if err != nil {
		t.Fatalf("ethclient dial: %v", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("chainID: %v", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		t.Fatalf("gasPrice: %v", err)
	}

	privKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		t.Fatalf("privKey: %v", err)
	}

	nonce, err := client.PendingNonceAt(ctx, me)
	if err != nil {
		t.Fatalf("nonce: %v", err)
	}

	// 1. Approve WETH → Morpho
	{
		// approve(address spender, uint256 amount)
		data := make([]byte, 4+32+32)
		copy(data[:4], approveSelector)
		copy(data[4+12:36], morphoBlue.Bytes())
		collateralAmount.FillBytes(data[36:68])

		t.Log("1. Approve WETH → Morpho")
		sendAndWait(t, client, ctx, nonce, gasPrice, chainID, weth, big.NewInt(0), data, privKey)
		nonce++
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

		sendAndWait(t, client, ctx, nonce, gasPrice, chainID, morphoBlue, big.NewInt(0), data, privKey)
		nonce++
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
		sendAndWait(t, client, ctx, nonce, gasPrice, chainID, morphoBlue, big.NewInt(0), data, privKey)
		nonce++
	}

	// Vérifier le solde USDC reçu
	{
		calldata := make([]byte, 4+32)
		copy(calldata[:4], balanceOfSelector)
		copy(calldata[4+12:], me.Bytes())

		result, err := client.CallContract(ctx, ethereum.CallMsg{
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
	amount := new(big.Int).Mul(market.LLTV, big.NewInt(2300))
	borrowAmount := new(big.Int).Div(amount, big.NewInt(1e12))

	a.ApproveAndBorrow(t, oneETH, borrowAmount)

}

func TestBorrowUSDCAndLiquidate(t *testing.T) {

	// Compte 1 d'Anvil comme liquidateur
	liquidator := common.HexToAddress(FundedAccounts[1])
	liquidatorKey := "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
	baseLiquidator := config.BaseLiquidatorUni
	a := StartAnvilFork(t, os.Getenv("BASE_HTTP_RPC_ALCH"), 0)

	oneETH := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18))

	// D'abord wrap ETH → WETH
	a.WrapETH(t, oneETH)

	// Emprunter le max soit 1ETH en USDC * LLTV  (6 décimales)
	amount := new(big.Int).Mul(market.Lltv, big.NewInt(2300))
	borrowAmount := new(big.Int).Div(amount, big.NewInt(1e12))

	a.ApproveAndBorrow(t, oneETH, borrowAmount)
	// juste tester la liquidation ici
	// liquidate consumer
	consumer := LiquidateConsumer(loadBaseTestConfig(a.RPCURL, a.WSURL))
	fee := big.NewInt(100)
	minOut := big.NewInt(1000)
	liqArg := liquidate.LiquidateArgs{
		MarketParams: market,
		Borrower:     common.HexToAddress(FundedAccounts[0]),
		SeizedAssets: utils.WAD,
		RepaidShares: big.NewInt(0),
		SwapRouter:   consumer.Config.Addresses.UniSwapRouter,
		PoolFee:      fee,
		MinOut:       minOut,
	}
	gasEst := ethereum.GasEstimator(ctx, msg)
	consumer.LiquidateCall(context.Background(), liqArg, gasEstimate)
}
