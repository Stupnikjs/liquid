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
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

var market = MarketParams{
	LoanToken:       usdc,
	CollateralToken: weth,
	Oracle:          common.HexToAddress("0xFEa2D58cEfCb9fcb597723c6bAE66fFE4193aFE4"),
	Irm:             common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687"),
	LLTV:            big.NewInt(860000000000000000),
}

type MarketParams struct {
	LoanToken       common.Address
	CollateralToken common.Address
	Oracle          common.Address
	Irm             common.Address
	LLTV            *big.Int
}

func encodeMarketParams(m MarketParams) []byte {
	buf := make([]byte, 5*32)
	copy(buf[12:32], m.LoanToken.Bytes())
	copy(buf[32+12:64], m.CollateralToken.Bytes())
	copy(buf[64+12:96], m.Oracle.Bytes())
	copy(buf[96+12:128], m.Irm.Bytes())
	m.LLTV.FillBytes(buf[128:160])
	return buf
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
