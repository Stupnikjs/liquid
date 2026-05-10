package testutil

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/connector"
	"github.com/Stupnikjs/liquid/internal/liquidate"
	"github.com/Stupnikjs/liquid/pkg/api"
	"github.com/Stupnikjs/liquid/pkg/config"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

var (
	anvil_pk0 = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	weth      = common.HexToAddress("0x4200000000000000000000000000000000000006")
	me        = common.HexToAddress(FundedAccounts[0])

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

var market = morpho.MarketContractParams{
	LoanToken:       usdc,
	CollateralToken: weth,
	Oracle:          common.HexToAddress("0xFEa2D58cEfCb9fcb597723c6bAE66fFE4193aFE4"),
	Irm:             common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687"),
	Lltv:            big.NewInt(860000000000000000),
}
var FundedAccounts = [10]string{
	"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
	"0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
	"0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC",
	"0x90F79bf6EB2c4f870365E785982E1f101E93b906",
	"0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65",
	"0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc",
	"0x976EA74026E726554dB657fA54763abd0C3a0aa9",
	"0x14dC79964da2C08b23698B3D3cc7Ca32193d9955",
	"0x23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f",
	"0xa0Ee7A142d267C1f36714E4a8F75612F20a79720",
}

type MarketParams struct {
	LoanToken       common.Address
	CollateralToken common.Address
	Oracle          common.Address
	Irm             common.Address
	LLTV            *big.Int
}

type txCtx struct {
	client   *ethclient.Client
	chainID  *big.Int
	gasPrice *big.Int
	privKey  *ecdsa.PrivateKey
	nonce    uint64
}

func (a *AnvilInstance) newTxCtx(t *testing.T) *txCtx {
	t.Helper()
	client, err := ethclient.Dial(a.RPCURL)
	if err != nil {
		t.Fatalf("ethclient dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	chainID, _ := client.ChainID(ctx)
	gasPrice, _ := client.SuggestGasPrice(ctx)
	privKey, _ := crypto.HexToECDSA(anvil_pk0)

	nonce, err := client.PendingNonceAt(ctx, me)
	if err != nil {
		t.Fatalf("nonce: %v", err)
	}
	return &txCtx{client, chainID, gasPrice, privKey, nonce}
}

// send transaction and wait for it
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

func encodeMarketParams(m morpho.MarketContractParams) []byte {
	buf := make([]byte, 5*32)
	copy(buf[12:32], m.LoanToken.Bytes())
	copy(buf[32+12:64], m.CollateralToken.Bytes())
	copy(buf[64+12:96], m.Oracle.Bytes())
	copy(buf[96+12:128], m.Irm.Bytes())
	m.Lltv.FillBytes(buf[128:160])
	return buf
}

func loadBaseTestConfig(anvilUrl, anvilWs string) config.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	chainid := int64(8453)
	signer, err := config.NewBaseSigner(chainid)
	if err != nil {
		fmt.Println(err)
	}

	return config.Config{
		Signer: signer,
		Addresses: config.Addresses{
			UniSwapRouter:      config.BaseUniswapV3Router,
			UniSwapQuoter:      config.BaseUniswapQuoterV2Addr,
			LiquidatorContract: config.BaseLiquidatorUni,
			Morpho:             config.MorphoMain,
			Wallet:             config.BaseWalletAddr,
		},
		ChainID: uint32(chainid),

		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: []string{anvilUrl, anvilUrl},
			WS:   []string{anvilWs, anvilWs},
		},
	}
}

func LiquidateConsumer(conf config.Config) *liquidate.Consumer {
	conn := connector.New(conf.RPC.HTTP[0], conf.RPC.HTTP[1], conf.RPC.WS[0])
	filter := api.MarketFilters{}
	mockMarketReader := cache.NewCache(conf, filter)
	return &liquidate.Consumer{
		Conn:   conn,
		Cache:  mockMarketReader.Markets,
		Config: conf,
	}
}
