package connector

import (
	"os/exec"
	"testing"
	"time"

	"github.com/lmittmann/w3"
)

type AnvilInstance struct {
	cmd    *exec.Cmd
	RPCURL string
	WSURL  string
}

// StartAnvil launches a local Anvil node and waits until it's ready.
// Calls t.Fatal if the process can't start.
func StartAnvil(t *testing.T) *AnvilInstance {
	t.Helper()
	cmd := exec.Command("anvil",
		"--port", "8545",
		"--block-time", "1", // auto-mine every second
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("anvil: failed to start: %v", err)
	}
	time.Sleep(500 * time.Millisecond) // wait for RPC to be ready

	a := &AnvilInstance{
		cmd:    cmd,
		RPCURL: "http://127.0.0.1:8545",
		WSURL:  "ws://127.0.0.1:8545",
	}
	t.Cleanup(a.Stop)
	return a
}

func (a *AnvilInstance) Stop() {
	if a.cmd.Process != nil {
		a.cmd.Process.Kill()
	}
}

// DialAnvil returns a w3.Client connected to the local Anvil node.
func (a *AnvilInstance) DialHTTP(t *testing.T) RPCClient {
	t.Helper()
	client, err := w3.Dial(a.RPCURL)
	if err != nil {
		t.Fatalf("anvil: dial HTTP failed: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func (a *AnvilInstance) DialWS(t *testing.T) RPCClient {
	t.Helper()
	client, err := w3.Dial(a.WSURL)
	if err != nil {
		t.Fatalf("anvil: dial WS failed: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}
