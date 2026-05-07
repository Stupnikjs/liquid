package connector

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3/w3types"
)

// ---------------------------------------------------------------------------
// Mock RPCClient
// ---------------------------------------------------------------------------

type mockRPCClient struct {
	mu        sync.Mutex
	callErr   error
	callCount int
	subErr    error
	subErrCh  chan error
}

func newMockClient() *mockRPCClient {
	return &mockRPCClient{
		subErrCh: make(chan error, 1),
	}
}

// just increment callCount
func (m *mockRPCClient) CallCtx(_ context.Context, calls ...w3types.RPCCaller) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	return m.callErr
}

func (m *mockRPCClient) Subscribe(_ w3types.RPCSubscriber) (*rpc.ClientSubscription, error) {
	if m.subErr != nil {
		return nil, m.subErr
	}
	// TODO: retourner une vraie *rpc.ClientSubscription si nécessaire
	return nil, nil
}

func (m *mockRPCClient) calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// ---------------------------------------------------------------------------
// Helper — connecteur de test
// ---------------------------------------------------------------------------

func newTestConnector(primary, fallback, ws RPCClient) *Connector {
	dialWS := func(_ string) (RPCClient, error) { return ws, nil }
	return newWithClients("primary", "fallback", "ws", primary, fallback, ws, dialWS)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestEthCallCtx_PrimarySuccess(t *testing.T) {
	primary := newMockClient()
	c := newTestConnector(primary, newMockClient(), newMockClient())

	if err := c.EthCallCtx(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if primary.calls() != 1 {
		t.Errorf("primary appelé %d fois, want 1", primary.calls())
	}
}

func TestEthCallCtx_FallbackOnRetryable(t *testing.T) {
	primary := newMockClient()
	primary.callErr = errors.New("EOF")
	fallback := newMockClient()
	c := newTestConnector(primary, fallback, newMockClient())

	if err := c.EthCallCtx(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fallback.calls() != 1 {
		t.Errorf("fallback appelé %d fois, want 1", fallback.calls())
	}
	if c.Metrics.Fallbacks.Load() != 1 {
		t.Errorf("Fallbacks metric = %d, want 1", c.Metrics.Fallbacks.Load())
	}
}

func TestEthCallCtx_RateLimited(t *testing.T) {
	c := newTestConnector(newMockClient(), newMockClient(), newMockClient())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // annulé immédiatement → limiter.Wait échoue

	if err := c.EthCallCtx(ctx, nil); !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestReconnectWS_Success(t *testing.T) {
	newWS := newMockClient()
	dialCount := 0
	dialWS := func(_ string) (RPCClient, error) {
		dialCount++
		return newWS, nil
	}
	c := newWithClients("p", "f", "ws", newMockClient(), newMockClient(), newMockClient(), dialWS)

	c.reconnectWS()

	if dialCount != 1 {
		t.Errorf("dialWS appelé %d fois, want 1", dialCount)
	}
	if c.Metrics.Reconnects.Load() != 1 {
		t.Errorf("Reconnects = %d, want 1", c.Metrics.Reconnects.Load())
	}
}

func TestWatchLogs_ExitsOnContextCancel(t *testing.T) {
	// Subscribe retourne une erreur immédiatement → watchLogs tente reconnect.
	// On annule le contexte via timeout pour forcer la sortie.
	ws := newMockClient()
	ws.subErr = errors.New("subscribe failed")

	dialWS := func(_ string) (RPCClient, error) { return ws, nil }
	c := newWithClients("p", "f", "ws", newMockClient(), newMockClient(), ws, dialWS)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		c.watchLogs(ctx, ethereum.FilterQuery{}, make(chan *types.Log, 10))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watchLogs ne s'est pas arrêté après l'annulation du contexte")
	}
}

func TestMetrics_Snapshot(t *testing.T) {
	c := newTestConnector(newMockClient(), newMockClient(), newMockClient())
	c.Metrics.EthCalls.Add(3)
	c.Metrics.Reconnects.Add(1)
	c.Metrics.Fallbacks.Add(2)

	calls, reconnects, fallbacks := c.Metrics.Snapshot()
	if calls != 3 || reconnects != 1 || fallbacks != 2 {
		t.Errorf("snapshot incorrect: (%d, %d, %d)", calls, reconnects, fallbacks)
	}
	// Vérifie le reset
	calls2, reconnects2, fallbacks2 := c.Metrics.Snapshot()
	if calls2 != 0 || reconnects2 != 0 || fallbacks2 != 0 {
		t.Error("les compteurs ne sont pas remis à zéro après Snapshot")
	}
}

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("EOF"), true},
		{errors.New("context deadline exceeded"), true},
		{errors.New("Request timeout"), true},
		{errors.New("execution reverted"), false},
	}
	for _, tc := range cases {
		if got := isRetryable(tc.err); got != tc.want {
			t.Errorf("isRetryable(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
