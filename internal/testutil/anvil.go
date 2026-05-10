package testutil

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"testing"
	"time"

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

// StartAnvil launches a local Anvil node, waits until its RPC is ready,
// and registers a cleanup that kills the process when the test ends.
// Calls t.Fatal if Anvil is not in PATH or fails to become ready.
func StartAnvil(t *testing.T) *AnvilInstance {
	t.Helper()
	return StartAnvilFork(t, "", 0)
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

// FundedAccounts returns the 10 pre-funded accounts Anvil creates by default.
// These are deterministic — same private keys every run.
// Index 0 is conventionally used as the "wallet" in tests.

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
