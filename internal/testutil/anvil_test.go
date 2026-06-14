package testutil

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"testing"

	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/internal/liquidate"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/swap"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

func TestWrapETH(t *testing.T) {
	a := StartAnvilFork(t, os.Getenv("BASE_HTTP_RPC_ALCH"), 0)
	oneETH := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18))
	a.WrapETH(t, anvil_pk0, oneETH)
}

func TestBorrowUSDC(t *testing.T) {

	a := StartAnvilFork(t, os.Getenv("BASE_HTTP_RPC_ALCH"), 0)

	oneETH := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18))

	// D'abord wrap ETH → WETH
	a.WrapETH(t, anvil_pk0, oneETH)

	// Emprunter le max soit 1ETH en USDC * LLTV  (6 décimales)
	amount := new(big.Int).Mul(market.LLTV, big.NewInt(1600))
	borrowAmount := new(big.Int).Div(amount, big.NewInt(1e12))

	a.ApproveAndBorrow(t, oneETH, borrowAmount)

}

func TestLiquidate(t *testing.T) {

	a := StartAnvilFork(t, os.Getenv("BASE_HTTP_RPC_ALCH"), 0)
	ctx := context.Background()

	txCtx, err := a.newTxCtx(t, os.Getenv("BASE_PK")[2:])
	privKey, _ := crypto.HexToECDSA(os.Getenv("BASE_PK")[2:])
	sender := crypto.PubkeyToAddress(privKey.PublicKey)
	t.Logf("sender: %s", sender.Hex())

	err = txCtx.client.Client().Call(nil, "anvil_setBalance", sender.Hex(), hexutil.EncodeBig(new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18))))
	if err != nil {
		t.Fatalf("anvil_setBalance: %v", err)
	}

	price := txCtx.PriceCall(ctx)
	a.LiquidationSetup(t, price)

	marketParam := *market.ToMarketContractParams()
	calldata, err := liquidate.BuildLiquidateCalldata(
		marketParam,
		common.HexToAddress(FundedAccounts[0]),
		utils.WAD, // seize all collateral
		big.NewInt(0),
		[]swap.SwapStep{buildWETHUSDCSwapStep()},
		big.NewInt(0),
	)
	if err != nil {
		t.Fatalf("EncodeLiquidateCalldata: %v", err)
	}
	// Drop oracle à 60%
	price60 := new(big.Int).Mul(price, big.NewInt(60))
	price60.Div(price60, big.NewInt(100))
	a.SetOraclePrice(t, txCtx.client, market.Oracle, price60)

	new_price := txCtx.PriceCall(ctx)
	t.Logf("new %s old %s", utils.FormatDecimals(new_price, 24), utils.FormatDecimals(price, 24))
	if new_price.Cmp(price) == 0 || new_price.Cmp(price) > 1 {
		t.Error("SetOraclePrice failed ")
	}
	txCtx.CallPosition(ctx)
	// avant sendAndWait
	gas, err := txCtx.client.EstimateGas(ctx, ethereum.CallMsg{
		From: sender,
		To:   &config.BaseLiquidatorLast,
		Data: calldata,
	})
	if err != nil {
		t.Fatalf("estimateGas: %v", err)
	}
	t.Logf("estimateGas: %d", gas)

	senderNonce, err := txCtx.client.PendingNonceAt(ctx, sender)
	if err != nil {
		t.Fatalf("nonce: %v", err)
	}
	t.Logf("%s usdc before", txCtx.CallTokenBallance(ctx, usdc, config.BaseLiquidatorLast).String())
	receipt := sendAndWait(
		t,
		txCtx.client,
		context.Background(),
		senderNonce,
		txCtx.gasPrice,
		txCtx.chainID,
		config.BaseLiquidatorLast, // to: le contrat liquidateur
		big.NewInt(0),             // value: 0 ETH
		calldata,                  // data: calldata encodé
		txCtx.privKey,
	)

	txCtx.CallPosition(ctx)
	t.Logf("liquidation txHash: %s, status: %d", receipt.TxHash, receipt.Status)
	t.Logf("%s usdc after", txCtx.CallTokenBallance(ctx, usdc, config.BaseLiquidatorLast).String())

}
