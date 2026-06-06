package connector

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
	"golang.org/x/time/rate"
)

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

// Connector is the public interface — no Morpho, no domain logic.
// Any bot (liquidation, arb, indexer) depends on this, not the concrete type.
type Connector interface {
	// CallCtx dispatches batch eth_calls on the primary (low-latency) RPC.
	// Use for time-critical calls: health factor checks, bundle submission.
	CallCtx(ctx context.Context, calls ...w3types.RPCCaller) error

	// SecondCallCtx dispatches batch eth_calls on the secondary RPC.
	// Use for non-critical calls: historical prices, backtesting, quotes.
	SecondCallCtx(ctx context.Context, calls ...w3types.RPCCaller) error

	// SubscribeLogs opens a WS log subscription for the given FilterQuery.
	// Returns a read-only channel; the subscription runs until ctx is cancelled.
	SubscribeLogs(ctx context.Context, query ethereum.FilterQuery) (<-chan *types.Log, error)

	// Metrics returns a point-in-time snapshot and resets all counters.
	Metrics() MetricsSnapshot

	// Close shuts down all underlying connections.
	Close() error
}

// RPCClient is the minimal surface needed from a w3.Client.
// Keeping it as an interface makes every method testable without a live endpoint.
type RPCClient interface {
	CallCtx(ctx context.Context, calls ...w3types.RPCCaller) error
	Subscribe(w3types.RPCSubscriber) (*rpc.ClientSubscription, error)
	Close() error
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// RPCEndpoints groups the three endpoint URLs by their role.
type RPCEndpoints struct {
	Primary string // fast/colocated node — liquidations, bundle submission
	Second  string // slower/cheaper node — quotes, historical reads, backtesting
	WS      string // WebSocket endpoint — log subscriptions
}

// ---------------------------------------------------------------------------
// Metrics
// ---------------------------------------------------------------------------

// ConnectorMetrics holds atomic counters readable from any goroutine.
type ConnectorMetrics struct {
	PrimaryCalls atomic.Uint64 // eth_calls dispatched on primary
	SecondCalls  atomic.Uint64 // eth_calls dispatched on second
	Reconnects   atomic.Uint64 // WS reconnection attempts
}

// MetricsSnapshot is a point-in-time copy returned to callers.
type MetricsSnapshot struct {
	PrimaryCalls uint64
	SecondCalls  uint64
	Reconnects   uint64
}

// snapshot resets all counters and returns their previous values.
func (m *ConnectorMetrics) snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		PrimaryCalls: m.PrimaryCalls.Swap(0),
		SecondCalls:  m.SecondCalls.Swap(0),
		Reconnects:   m.Reconnects.Swap(0),
	}
}

// ---------------------------------------------------------------------------
// EthConnector — concrete implementation
// ---------------------------------------------------------------------------

type EthConnector struct {
	mu        sync.RWMutex
	primary   RPCClient // primary is for garbage calls
	second    RPCClient // second is main
	ws        RPCClient
	dialWS    func(url string) (RPCClient, error)
	endpoints RPCEndpoints
	limiter   *rate.Limiter
	metrics   ConnectorMetrics
}

// New dials all three endpoints. Panics on failure — connectivity is a hard
// requirement at bot startup.
func New(endpoints RPCEndpoints) *EthConnector {
	dial := func(url string) (RPCClient, error) {
		return w3.Dial(url)
	}

	primary, err := dial(endpoints.Primary)
	if err != nil {
		panic(fmt.Sprintf("connector: primary dial failed: %v", err))
	}
	second, err := dial(endpoints.Second)
	if err != nil {
		panic(fmt.Sprintf("connector: second dial failed: %v", err))
	}
	ws, err := dial(endpoints.WS)
	if err != nil {
		panic(fmt.Sprintf("connector: ws dial failed: %v", err))
	}

	return &EthConnector{
		primary:   primary,
		second:    second,
		ws:        ws,
		dialWS:    dial,
		endpoints: endpoints,
		limiter:   rate.NewLimiter(rate.Every(time.Minute/500), 15),
	}
}

// newWithClients is the test constructor — accepts pre-built RPCClient mocks.
func newWithClients(
	endpoints RPCEndpoints,
	primary, second, ws RPCClient,
	dialWS func(string) (RPCClient, error),
) *EthConnector {
	return &EthConnector{
		primary:   primary,
		second:    second,
		ws:        ws,
		dialWS:    dialWS,
		endpoints: endpoints,
		limiter:   rate.NewLimiter(rate.Every(time.Minute/500), 15),
	}
}

// ---------------------------------------------------------------------------
// Connector interface implementation
// ---------------------------------------------------------------------------

// CallCtx dispatches calls on the primary (low-latency) RPC.
func (c *EthConnector) CallCtx(ctx context.Context, calls ...w3types.RPCCaller) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("%v", err)
	}
	c.mu.RLock()
	primary := c.primary
	c.mu.RUnlock()

	c.metrics.PrimaryCalls.Add(uint64(len(calls)))
	return primary.CallCtx(ctx, calls...)
}

// SecondCallCtx dispatches calls on the secondary (non-critical) RPC.
func (c *EthConnector) SecondCallCtx(ctx context.Context, calls ...w3types.RPCCaller) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("%v", err)
	}
	c.mu.RLock()
	second := c.second
	c.mu.RUnlock()

	c.metrics.SecondCalls.Add(uint64(len(calls)))
	return second.CallCtx(ctx, calls...)
}

// SubscribeLogs opens a WS log subscription for the given query.
// The returned channel is closed when ctx is cancelled.
func (c *EthConnector) SubscribeLogs(ctx context.Context, query ethereum.FilterQuery) (<-chan *types.Log, error) {
	ch := make(chan *types.Log, 100)
	go c.watchLogs(ctx, query, ch)
	return ch, nil
}

// Metrics returns a snapshot of current counters and resets them to zero.
func (c *EthConnector) Metrics() MetricsSnapshot {
	return c.metrics.snapshot()
}

func (c *EthConnector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error

	if err := c.primary.Close(); err != nil {
		errs = append(errs, fmt.Errorf("primary close failed: %w", err))
	}

	if err := c.second.Close(); err != nil {
		errs = append(errs, fmt.Errorf("second close failed: %w", err))
	}

	if err := c.ws.Close(); err != nil {
		errs = append(errs, fmt.Errorf("ws close failed: %w", err))
	}

	return errors.Join(errs...)
}

// ---------------------------------------------------------------------------
// Internal — WS management
// ---------------------------------------------------------------------------

func (c *EthConnector) getWS() RPCClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ws
}

func (c *EthConnector) reconnectWS() {
	for {
		log.Printf("[connector] reconnecting WS to %s", c.endpoints.WS)
		client, err := c.dialWS(c.endpoints.WS)
		if err != nil {
			log.Printf("[connector] WS reconnect failed: %v — retrying in 2s", err)
			time.Sleep(2 * time.Second)
			continue
		}
		c.mu.Lock()
		c.ws = client
		c.mu.Unlock()
		c.metrics.Reconnects.Add(1)
		log.Println("[connector] WS reconnected")
		return
	}
}

func (c *EthConnector) watchLogs(ctx context.Context, query ethereum.FilterQuery, ch chan *types.Log) {
	defer close(ch)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		sub, err := c.getWS().Subscribe(eth.NewLogs(ch, query))
		if err != nil {
			log.Printf("[connector] subscribe failed: %v — reconnecting", err)
			c.reconnectWS()
			continue
		}
		log.Println("[connector] subscribed to logs")

		select {
		case err := <-sub.Err():
			log.Printf("[connector] subscription error: %v — reconnecting", err)
			sub.Unsubscribe()
			c.reconnectWS()
		case <-ctx.Done():
			sub.Unsubscribe()
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
