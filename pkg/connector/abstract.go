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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
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

	CallCtx(ctx context.Context, calls []rpc.BatchElem) error

	// SecondCallCtx dispatches batch eth_calls on the secondary RPC.
	// Use for non-critical calls: historical prices, backtesting, quotes.
	SecondCallCtx(ctx context.Context, calls []rpc.BatchElem) error

	SendRawTx(ctx context.Context, tx *types.Transaction) (common.Hash, error)
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
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
	EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error)
	Close()
}

// rpcClientAdapter bridges *rpc.Client (which has Close() with no return value)
// to the RPCClient interface.
type rpcClientAdapter struct {
	c *rpc.Client
}

// dialRPC dials any endpoint (HTTP, HTTPS, WS, WSS) via go-ethereum's rpc package.
func dialRPC(ctx context.Context, url string) (RPCClient, error) {
	c, err := rpc.DialContext(ctx, url)

	if err != nil {
		return nil, err
	}
	return &rpcClientAdapter{c}, nil
}

func (a *rpcClientAdapter) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return a.c.CallContext(ctx, result, method, args...)
}

func (a *rpcClientAdapter) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	return a.c.BatchCallContext(ctx, b)
}

func (a *rpcClientAdapter) EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error) {
	return a.c.EthSubscribe(ctx, channel, args...)
}

func (a *rpcClientAdapter) Close() {
	a.c.Close()
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
// Error classification
// ---------------------------------------------------------------------------

var ErrRateLimited = errors.New("connector: rate limit exceeded")

var transientErrors = []string{
	"Request timeout",
	"context deadline exceeded",
	"EOF",
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, s := range transientErrors {
		if stringContains(msg, s) {
			return true
		}
	}
	return false
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
	primary   RPCClient
	second    RPCClient
	ws        RPCClient
	dialWS    func(url string) (RPCClient, error)
	endpoints RPCEndpoints
	limiter   *rate.Limiter
	metrics   ConnectorMetrics
}

// New dials all three endpoints. Panics on failure — connectivity is a hard
// requirement at bot startup.
// New dials all three endpoints. Panics on failure — connectivity is a hard
// requirement at bot startup.
func New(endpoints RPCEndpoints) *EthConnector {
	ctx := context.Background()

	dial := func(url string) (RPCClient, error) {
		return dialRPC(ctx, url)
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

func (c *EthConnector) SendRawTx(ctx context.Context, tx *types.Transaction) (common.Hash, error) {
	data, err := tx.MarshalBinary()
	if err != nil {
		return common.Hash{}, err
	}
	var hash common.Hash
	err = c.primary.CallContext(ctx, &hash, "eth_sendRawTransaction", hexutil.Encode(data))
	return hash, err
}

// CallCtx dispatches a JSON-RPC batch on the primary (low-latency) client.
// Each BatchElem.Error must be checked individually after the call returns nil.
func (c *EthConnector) CallCtx(ctx context.Context, calls []rpc.BatchElem) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrRateLimited, err)
	}
	c.mu.RLock()
	primary := c.primary
	c.mu.RUnlock()

	c.metrics.PrimaryCalls.Add(uint64(len(calls)))
	return primary.BatchCallContext(ctx, calls)
}

// SecondCallCtx dispatches a JSON-RPC batch on the secondary (non-critical) client.
func (c *EthConnector) SecondCallCtx(ctx context.Context, calls []rpc.BatchElem) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrRateLimited, err)
	}
	c.mu.RLock()
	second := c.second
	c.mu.RUnlock()

	c.metrics.SecondCalls.Add(uint64(len(calls)))
	return second.BatchCallContext(ctx, calls)
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

	c.primary.Close()
	c.second.Close()
	c.ws.Close()

	return nil
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

	filterArg := map[string]interface{}{
		"address": query.Addresses,
		"topics":  query.Topics,
	}
	if query.FromBlock != nil {
		filterArg["fromBlock"] = hexutil.EncodeBig(query.FromBlock)
	}
	if query.ToBlock != nil {
		filterArg["toBlock"] = hexutil.EncodeBig(query.ToBlock)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		sub, err := c.getWS().EthSubscribe(ctx, ch, "logs", filterArg)
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
