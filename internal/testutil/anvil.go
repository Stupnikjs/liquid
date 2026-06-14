package testutil

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/lmittmann/w3"
)

// AnvilInstance holds a running Anvil process and its RPC endpoints.
// Always obtained through StartAnvil — never constructed directly.
type AnvilInstance struct {
	cmd    *exec.Cmd
	Port   string
	RPCURL string
	WSURL  string
}

// StartAnvilFork launches Anvil forked from forkURL at the given block number.
// Pass an empty forkURL to start a blank chain. Pass 0 for forkBlock to fork
// at the latest block.
func StartAnvilFork(t *testing.T, forkURL string, forkBlock uint64) *AnvilInstance {
	t.Helper()

	port := "18545" // offset from default to avoid colliding with a running node
	args := []string{
		"--port", port,
		"--block-time", "1", // auto-mine every second
		"--silent", // suppress Anvil's own log output in test runs
	}
	if forkURL != "" {
		args = append(args, "--fork-url", forkURL)
		if forkBlock > 0 {
			args = append(args, "--fork-block-number", fmt.Sprintf("%d", forkBlock))
		}
	}

	cmd := exec.Command("anvil", args...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("testutil/anvil: failed to start (is anvil in PATH?): %v", err)
	}

	a := &AnvilInstance{
		cmd:    cmd,
		Port:   port,
		RPCURL: fmt.Sprintf("http://127.0.0.1:%s", port),
		WSURL:  fmt.Sprintf("ws://127.0.0.1:%s", port),
	}

	t.Cleanup(a.stop)

	if err := a.waitReady(5 * time.Second); err != nil {
		t.Fatalf("testutil/anvil: node did not become ready: %v", err)
	}

	return a
}

// stop kills the Anvil process. Called automatically via t.Cleanup.
func (a *AnvilInstance) stop() {
	if a.cmd != nil && a.cmd.Process != nil {
		_ = a.cmd.Process.Kill()
		_ = a.cmd.Wait() // reap the zombie
	}
}

// waitReady polls the HTTP endpoint until it responds or the timeout expires.
func (a *AnvilInstance) waitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Post(
			a.RPCURL,
			"application/json",
			nil,
		)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout after %s waiting for %s", timeout, a.RPCURL)
}

// DialHTTP returns a w3 client connected to the Anvil HTTP endpoint.
// The client is closed automatically when the test ends.
func (a *AnvilInstance) DialHTTP(t *testing.T) *w3.Client {
	t.Helper()
	client, err := w3.Dial(a.RPCURL)
	if err != nil {
		t.Fatalf("testutil/anvil: HTTP dial failed: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// DialWS returns a w3 client connected to the Anvil WebSocket endpoint.
// The client is closed automatically when the test ends.
func (a *AnvilInstance) DialWS(t *testing.T) *w3.Client {
	t.Helper()
	client, err := w3.Dial(a.WSURL)
	if err != nil {
		t.Fatalf("testutil/anvil: WS dial failed: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// WaitForBlock blocks until Anvil has mined at least n blocks beyond the
// current head, or ctx is cancelled. Useful for tests that need confirmations.
func WaitForBlock(ctx context.Context, client *w3.Client, n uint64) error {
	// implementation depends on w3 block number fetch — placeholder
	// replace with: eth.BlockNumber + polling loop
	timer := time.NewTimer(time.Duration(n+1) * time.Second) // 1s block time
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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
		From:  crypto.PubkeyToAddress(privKey.PublicKey),
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
	_, err = client.CallContract(ctx, ethereum.CallMsg{
		From:  crypto.PubkeyToAddress(privKey.PublicKey),
		To:    &to,
		Value: value,
		Data:  data,
	}, nil)

	if err != nil {
		t.Fatalf("simulation failed: %v", err)
	}

	if err := client.SendTransaction(ctx, signed); err != nil {
		t.Fatalf("sendTransaction: %v", err)
	}

	t.Logf("txHash: %s", signed.Hash())

	var receipt *types.Receipt
	for i := 0; i < 60; i++ {
		time.Sleep(500 * time.Millisecond)
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
