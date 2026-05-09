package testutil

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

var (
	weth = common.HexToAddress("0x4200000000000000000000000000000000000006")
	me   = common.HexToAddress(FundedAccounts[0])

	usdc       = common.HexToAddress("0x833589fCD6EDB6E08f4c7C32D4f71b54bdA02913")
	morphoBlue = common.HexToAddress("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb")

	// approve(address,uint256)
	approveSelector = []byte{0x09, 0x5e, 0xa7, 0xb3}

	// supplyCollateral((address,address,address,address,uint256),uint256,address,bytes)
	supplyCollateralSelector = []byte{0x23, 0x8d, 0x65, 0x79}

	// borrow((address,address,address,address,uint256),uint256,uint256,address,address)
	borrowSelector = []byte{0x50, 0xd8, 0xcd, 0x4b}

	balanceOfSelector = []byte{0x70, 0xa0, 0x82, 0x31}
)

// depositData retourne le calldata de deposit() : sélecteur 0xd0e30db0
var depositCalldata = []byte{0xd0, 0xe3, 0x0d, 0xb0}

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

var market = MarketParams{
	LoanToken:       usdc,
	CollateralToken: weth,
	Oracle:          common.HexToAddress("0x2DC205F24BCb6B311E5cdf0745B0741648Aebd3d"),
	Irm:             common.HexToAddress("0x4647B8FfC145fF3D82BDAf0222B869bDAa6072bE"),
	LLTV:            big.NewInt(860000000000000000),
}

type MarketParams struct {
	LoanToken       common.Address
	CollateralToken common.Address
	Oracle          common.Address
	Irm             common.Address
	LLTV            *big.Int
}

// encodeMarketParams encode la struct MarketParams en ABI (5 × 32 bytes)
func encodeMarketParams(m MarketParams) []byte {
	buf := make([]byte, 5*32)
	copy(buf[12:32], m.LoanToken.Bytes())
	copy(buf[32+12:64], m.CollateralToken.Bytes())
	copy(buf[64+12:96], m.Oracle.Bytes())
	copy(buf[96+12:128], m.Irm.Bytes())
	m.LLTV.FillBytes(buf[128:160])
	return buf
}

func sendAndWait(t *testing.T, client *ethclient.Client, ctx context.Context,
	nonce uint64, gasPrice *big.Int, chainID *big.Int,
	to common.Address, value *big.Int, data []byte,
	privKey *ecdsa.PrivateKey) *types.Receipt {

	t.Helper()

	gas, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From:  me,
		To:    &to,
		Value: value,
		Data:  data,
	})
	if err != nil {
		t.Fatalf("estimateGas: %v", err)
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    value,
		Gas:      gas,
		GasPrice: gasPrice,
		Data:     data,
	})

	signer := types.LatestSignerForChainID(chainID)
	signed, err := types.SignTx(tx, signer, privKey)
	if err != nil {
		t.Fatalf("signTx: %v", err)
	}

	if err := client.SendTransaction(ctx, signed); err != nil {
		t.Fatalf("sendTransaction: %v", err)
	}

	t.Logf("txHash: %s", signed.Hash())

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
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("tx revertée, status: %d", receipt.Status)
	}
	return receipt
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
		copy(calldata[:4], []byte{0x70, 0xa0, 0x82, 0x31})
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

	// Emprunter 100 USDC (6 décimales)
	borrowAmount := new(big.Int).Mul(big.NewInt(100), big.NewInt(1e6))
	a.ApproveAndBorrow(t, oneETH, borrowAmount)
}
