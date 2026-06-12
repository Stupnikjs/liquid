package testutil

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/lmittmann/w3"
)

// AnvilOptions configure le comportement du nœud de test.
type AnvilOptions struct {
	ForkURL   string
	ForkBlock uint64
	// BlockTime 0 = mine-on-demand (recommandé pour les tests de replay)
	// BlockTime > 0 = auto-mine toutes les N secondes
	BlockTime uint64
	Silent    bool
}

type AnvilInstance struct {
	cmd    *exec.Cmd
	Port   string
	RPCURL string
	WSURL  string
	// mineOnDemand est vrai quand BlockTime == 0
	// sendAndWait appelle Mine(1) explicitement dans ce cas
	mineOnDemand bool
}

// StartAnvilFork est le point d'entrée principal.
// forkURL vide = chain vierge. forkBlock 0 = dernier bloc.
func StartAnvilFork(t *testing.T, forkURL string, forkBlock uint64) *AnvilInstance {
	t.Helper()
	return StartAnvil(t, AnvilOptions{
		ForkURL:   forkURL,
		ForkBlock: forkBlock,
		Silent:    true,
		// BlockTime intentionnellement absent → mine-on-demand
	})
}

func StartAnvil(t *testing.T, opts AnvilOptions) *AnvilInstance {
	t.Helper()

	// Port dynamique — élimine les collisions entre tests parallèles
	port, err := freePort()
	if err != nil {
		t.Fatalf("testutil/anvil: no free port: %v", err)
	}

	args := []string{
		"--port", port,
		"--gas-price", "0", // pas de friction gas dans les tests
	}

	if opts.BlockTime > 0 {
		args = append(args, "--block-time", fmt.Sprintf("%d", opts.BlockTime))
	}
	// BlockTime == 0 → pas de flag → Anvil mine à la demande

	if opts.Silent {
		args = append(args, "--silent")
	}

	if opts.ForkURL != "" {
		args = append(args, "--fork-url", opts.ForkURL)
		if opts.ForkBlock > 0 {
			args = append(args, "--fork-block-number",
				fmt.Sprintf("%d", opts.ForkBlock))
		}
	}

	cmd := exec.Command("anvil", args...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("testutil/anvil: failed to start (is anvil in PATH?): %v", err)
	}

	a := &AnvilInstance{
		cmd:          cmd,
		Port:         port,
		RPCURL:       fmt.Sprintf("http://127.0.0.1:%s", port),
		WSURL:        fmt.Sprintf("ws://127.0.0.1:%s", port),
		mineOnDemand: opts.BlockTime == 0,
	}

	t.Cleanup(a.stop)

	if err := a.waitReady(10 * time.Second); err != nil {
		t.Fatalf("testutil/anvil: node did not become ready: %v", err)
	}

	return a
}

func freePort() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer ln.Close()
	return fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port), nil
}

func (a *AnvilInstance) stop() {
	if a.cmd != nil && a.cmd.Process != nil {
		_ = a.cmd.Process.Kill()
		_ = a.cmd.Wait()
	}
}

// waitReady envoie un vrai JSON-RPC pour vérifier qu'Anvil répond
func (a *AnvilInstance) waitReady(timeout time.Duration) error {
	body := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Post(a.RPCURL, "application/json",
			strings.NewReader(body))
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout after %s waiting for %s", timeout, a.RPCURL)
}

// ── JSON-RPC bas niveau ───────────────────────────────────────────────────────

func (a *AnvilInstance) rpcCall(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	payload, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, a.RPCURL,
		strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("rpc %s: %s", method, result.Error.Message)
	}
	return result.Result, nil
}

// ── Contrôle du temps et des blocs ───────────────────────────────────────────

func (a *AnvilInstance) Mine(ctx context.Context, blocks uint64) error {
	_, err := a.rpcCall(ctx, "anvil_mine", hexutil.EncodeUint64(blocks))
	return err
}

// IncreaseTime avance le timestamp de N secondes.
// N'a d'effet on-chain qu'au prochain bloc miné.
func (a *AnvilInstance) IncreaseTime(ctx context.Context, seconds uint64) error {
	_, err := a.rpcCall(ctx, "evm_increaseTime", hexutil.EncodeUint64(seconds))
	return err
}

// SetNextBlockTimestamp fixe le timestamp exact du prochain bloc.
func (a *AnvilInstance) SetNextBlockTimestamp(ctx context.Context, ts uint64) error {
	_, err := a.rpcCall(ctx, "evm_setNextBlockTimestamp", hexutil.EncodeUint64(ts))
	return err
}

// FastForward avance de N blocs avec des timestamps cohérents (12s/bloc).
// Morpho Blue calcule les intérêts via block.timestamp — c'est cette fonction
// qu'il faut appeler pour simuler l'accumulation des intérêts.
func (a *AnvilInstance) FastForward(ctx context.Context, blocks uint64) error {
	const blockTime = 12

	raw, err := a.rpcCall(ctx, "eth_getBlockByNumber", "latest", false)
	if err != nil {
		return err
	}
	var block struct {
		Timestamp hexutil.Uint64 `json:"timestamp"`
	}
	if err := json.Unmarshal(raw, &block); err != nil {
		return err
	}

	if err := a.IncreaseTime(ctx, blocks*blockTime); err != nil {
		return err
	}
	return a.Mine(ctx, blocks)
}

// FastForwardDays est un raccourci pour simuler N jours d'intérêts Morpho.
func (a *AnvilInstance) FastForwardDays(ctx context.Context, days uint64) error {
	return a.FastForward(ctx, days*7200) // ~7200 blocs/jour sur mainnet
}

// ── Manipulation d'état ───────────────────────────────────────────────────────

func (a *AnvilInstance) Snapshot(ctx context.Context) (string, error) {
	raw, err := a.rpcCall(ctx, "evm_snapshot")
	if err != nil {
		return "", err
	}
	var snap string
	return snap, json.Unmarshal(raw, &snap)
}

func (a *AnvilInstance) Revert(ctx context.Context, snap string) error {
	_, err := a.rpcCall(ctx, "evm_revert", snap)
	return err
}

func (a *AnvilInstance) SetBalance(ctx context.Context, addr common.Address, amount *big.Int) error {
	_, err := a.rpcCall(ctx, "anvil_setBalance",
		addr.Hex(), hexutil.EncodeBig(amount))
	return err
}

func (a *AnvilInstance) SetStorageAt(ctx context.Context, addr common.Address, slot, value common.Hash) error {
	_, err := a.rpcCall(ctx, "anvil_setStorageAt",
		addr.Hex(), slot.Hex(), value.Hex())
	return err
}

func (a *AnvilInstance) ImpersonateAccount(ctx context.Context, addr common.Address) error {
	_, err := a.rpcCall(ctx, "anvil_impersonateAccount", addr.Hex())
	return err
}

// ── Clients w3 ───────────────────────────────────────────────────────────────

func (a *AnvilInstance) DialHTTP(t *testing.T) *w3.Client {
	t.Helper()
	client, err := w3.Dial(a.RPCURL)
	if err != nil {
		t.Fatalf("testutil/anvil: HTTP dial failed: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func (a *AnvilInstance) DialWS(t *testing.T) *w3.Client {
	t.Helper()
	client, err := w3.Dial(a.WSURL)
	if err != nil {
		t.Fatalf("testutil/anvil: WS dial failed: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// WaitForBlock poll réel sur eth_blockNumber
func (a *AnvilInstance) WaitForBlock(ctx context.Context, target uint64) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		raw, err := a.rpcCall(ctx, "eth_blockNumber")
		if err != nil {
			return err
		}
		var hexNum string
		if err := json.Unmarshal(raw, &hexNum); err != nil {
			return err
		}
		current, err := hexutil.DecodeUint64(hexNum)
		if err != nil {
			return err
		}
		if current >= target {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// ── txCtx ─────────────────────────────────────────────────────────────────────

type MarketParams struct {
	LoanToken       common.Address
	CollateralToken common.Address
	Oracle          common.Address
	Irm             common.Address
	LLTV            *big.Int
}

type txCtx struct {
	client  *ethclient.Client
	chainID *big.Int
	privKey *ecdsa.PrivateKey
	nonce   uint64
}

func (a *AnvilInstance) NewTxCtx(t *testing.T, privkeyStr string) *txCtx {
	t.Helper()

	client, err := ethclient.Dial(a.RPCURL)
	if err != nil {
		t.Fatalf("ethclient dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("chainID: %v", err)
	}

	privKey, err := crypto.HexToECDSA(privkeyStr)
	if err != nil {
		t.Fatalf("privkey: %v", err)
	}

	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, addr)
	if err != nil {
		t.Fatalf("nonce: %v", err)
	}

	// gasPrice inutile : on a lancé Anvil avec --gas-price 0
	return &txCtx{client, chainID, privKey, nonce}
}

// SendAndWait signe, envoie, mine si nécessaire, et retourne le receipt.
// Incrémente le nonce automatiquement.
func (a *AnvilInstance) SendAndWait(t *testing.T, ctx context.Context, tc *txCtx,
	to common.Address, value *big.Int, data []byte) *types.Receipt {

	t.Helper()

	from := crypto.PubkeyToAddress(tc.privKey.PublicKey)

	gas, err := tc.client.EstimateGas(ctx, ethereum.CallMsg{
		From:  from,
		To:    &to,
		Value: value,
		Data:  data,
	})
	if err != nil {
		t.Fatalf("estimateGas: %v", err)
	}
	gas = gas * 120 / 100 // marge +20%

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    tc.nonce,
		To:       &to,
		Value:    value,
		Gas:      gas,
		GasPrice: big.NewInt(0), // --gas-price 0 sur Anvil
		Data:     data,
	})

	signer := types.LatestSignerForChainID(tc.chainID)
	signed, err := types.SignTx(tx, signer, tc.privKey)
	if err != nil {
		t.Fatalf("signTx: %v", err)
	}

	if err := tc.client.SendTransaction(ctx, signed); err != nil {
		t.Fatalf("sendTransaction: %v", err)
	}

	tc.nonce++

	// En mine-on-demand : mine explicitement pour inclure la tx
	if a.mineOnDemand {
		if err := a.Mine(ctx, 1); err != nil {
			t.Fatalf("mine: %v", err)
		}
	}

	receipt, err := tc.client.TransactionReceipt(ctx, signed.Hash())
	if err != nil {
		t.Fatalf("receipt %s: %v", signed.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("tx revertée: %s", signed.Hash())
	}

	return receipt
}