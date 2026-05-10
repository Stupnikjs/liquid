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

func (a *AnvilInstance) newTxCtx(t *testing.T, privkeyStr string) (*txCtx, error) {
	t.Helper()
	client, err := ethclient.Dial(a.RPCURL)
	if err != nil {
		t.Fatalf("ethclient dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	privKey, err := crypto.HexToECDSA(privkeyStr)
	if err != nil {
		return nil, err
	}

	nonce, err := client.PendingNonceAt(ctx, me)
	if err != nil {
		return nil, err
	}
	return &txCtx{client, chainID, gasPrice, privKey, nonce}, nil
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
