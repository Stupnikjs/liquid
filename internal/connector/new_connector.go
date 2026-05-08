package connector

import (
	"context"
	"errors"
	"fmt"
	"log"

	"time"

	"sync"
	"sync/atomic"

	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/config"
	"golang.org/x/time/rate"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

// RPCClient is the minimal surface the Connector needs from a w3.Client.
// Using an interface makes every method testable without a live RPC endpoint.
type RPCClient interface {
	CallCtx(ctx context.Context, calls ...w3types.RPCCaller) error
	Subscribe(w3types.RPCSubscriber) (*rpc.ClientSubscription, error)
}

// ---------------------------------------------------------------------------
// Metrics — observable, independent of logging
// ---------------------------------------------------------------------------

// ConnectorMetrics holds atomic counters that any caller can read at any time
// (health-check endpoint, dashboard, periodic log flush — all independent).
type ConnectorMetrics struct {
	EthCalls   atomic.Uint64 // incremented per RPC call dispatched
	Reconnects atomic.Uint64 // incremented each time the WS is re-dialled
	Fallbacks  atomic.Uint64 // incremented each time the fallback HTTP is used
}

// Snapshot returns a point-in-time copy and resets all counters to zero.
// Safe to call from any goroutine.
func (m *ConnectorMetrics) Snapshot() (calls, reconnects, fallbacks uint64) {
	calls = m.EthCalls.Swap(0)
	reconnects = m.Reconnects.Swap(0)
	fallbacks = m.Fallbacks.Swap(0)
	return
}

// ---------------------------------------------------------------------------
// Retryable error classification
// ---------------------------------------------------------------------------

// retryableErrors is the list of substrings that indicate a transient HTTP
// failure worth retrying on the fallback. Centralised here so a string change
// in the upstream library is caught in one place.
var retryableErrors = []string{
	"Request timeout",
	"context deadline exceeded",
	"EOF",
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, s := range retryableErrors {
		if contains(msg, s) {
			return true
		}
	}
	return false
}

// sentinel so callers can errors.Is() on a rate-limit refusal
var ErrRateLimited = errors.New("connector: rate limit exceeded")

// ---------------------------------------------------------------------------
// Connector
// ---------------------------------------------------------------------------

type Connector struct {
	mu sync.RWMutex

	// transport layer — unexported, swapped only through reconnectWS
	primary  RPCClient
	fallback RPCClient
	ws       RPCClient

	// dialWS is injectable: production code passes w3.Dial, tests pass a mock
	dialWS func(url string) (RPCClient, error)

	// RPC URLs kept for reconnect
	mainRPC  string
	quoteRPC string
	wsRPC    string

	// logsCh is internal; expose read-only to consumers via LogsCh()
	logsCh chan *types.Log

	limiter *rate.Limiter
	Metrics ConnectorMetrics
}

// New builds a production Connector. It dials all three endpoints and panics
// if any dial fails — same behaviour as before, intentional for a bot where
// connectivity is a hard requirement at startup.
func New(mainRPC, quoteRPC, wsRPC string) *Connector {
	dial := func(url string) (RPCClient, error) {
		return w3.Dial(url)
	}

	primary, err := dial(mainRPC)
	if err != nil {
		panic(fmt.Sprintf("connector: primary dial failed: %v", err))
	}
	fallback, err := dial(quoteRPC)
	if err != nil {
		panic(fmt.Sprintf("connector: fallback dial failed: %v", err))
	}
	ws, err := dial(wsRPC)
	if err != nil {
		panic(fmt.Sprintf("connector: ws dial failed: %v", err))
	}

	return &Connector{
		primary:  primary,
		fallback: fallback,
		ws:       ws,
		dialWS:   dial,
		mainRPC:  mainRPC,
		quoteRPC: quoteRPC,
		wsRPC:    wsRPC,
		logsCh:   make(chan *types.Log, 100),
		limiter:  rate.NewLimiter(rate.Every(time.Minute/300), 10),
	}
}

// newWithClients is the test constructor — accepts pre-built RPCClient mocks.
func newWithClients(mainRPC, quoteRPC, wsRPC string,
	primary, fallback, ws RPCClient,
	dialWS func(string) (RPCClient, error),
) *Connector {
	return &Connector{
		primary:  primary,
		fallback: fallback,
		ws:       ws,
		dialWS:   dialWS,
		mainRPC:  mainRPC,
		quoteRPC: quoteRPC,
		wsRPC:    wsRPC,
		logsCh:   make(chan *types.Log, 100),
		limiter:  rate.NewLimiter(rate.Every(time.Minute/300), 10),
	}
}

// ---------------------------------------------------------------------------
// Public surface
// ---------------------------------------------------------------------------

// LogsCh returns the read-only end of the internal log channel.
// Consumers (runner, cache) receive events without being able to send or close.
func (c *Connector) LogsCh() <-chan *types.Log {
	return c.logsCh
}

// EthCallCtx dispatches a batch of eth_calls through the primary HTTP client.
// On a transient/timeout error it automatically retries on the fallback and
// increments the Fallbacks counter. Rate-limit refusals wrap ErrRateLimited.
func (c *Connector) EthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrRateLimited, err)
	}

	c.mu.RLock()
	primary := c.primary
	fallback := c.fallback
	c.mu.RUnlock()

	c.Metrics.EthCalls.Add(uint64(len(calls)))
	err := primary.CallCtx(ctx, calls...)
	if err != nil && isRetryable(err) {
		c.Metrics.Fallbacks.Add(1)
		return fallback.CallCtx(ctx, calls...)
	}
	return err
}

// using alchemy as fallback for now
func (c *Connector) FallBackEthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrRateLimited, err)
	}

	c.mu.RLock()
	primary := c.primary
	fallback := c.fallback
	c.mu.RUnlock()

	c.Metrics.Fallbacks.Add(uint64(len(calls)))
	err := fallback.CallCtx(ctx, calls...)
	if err != nil && isRetryable(err) {
		c.Metrics.EthCalls.Add(1)
		return primary.CallCtx(ctx, calls...)
	}
	return err
}

// SubscribeToEventPos starts the WS log subscription for all Morpho position
// events. It blocks until ctx is cancelled.
func (c *Connector) SubscribeToEventPos(ctx context.Context, conf config.Config) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{conf.Addresses.Morpho},
		Topics: [][]common.Hash{{
			config.EventBorrow.Topic0,
			config.EventRepay.Topic0,
			config.EventSupplyCollateral.Topic0,
			config.EventLiquidate.Topic0,
			config.EventAccrueInterest.Topic0,
		}},
	}
	c.watchLogs(ctx, query, c.logsCh)
}

// LogMetrics periodically snapshots and emits metrics into logChan.
// Runs until ctx is cancelled.
func (c *Connector) LogMetrics(ctx context.Context, logChan chan string) {
	utils.RunTicker(ctx, 10*time.Minute, func() {
		calls, reconnects, fallbacks := c.Metrics.Snapshot()
		logChan <- fmt.Sprintf(
			"[connector] eth_calls=%d reconnects=%d fallbacks=%d\n",
			calls, reconnects, fallbacks,
		)
	})
}

// ---------------------------------------------------------------------------
// Internal — WS management
// ---------------------------------------------------------------------------

func (c *Connector) getWSClient() RPCClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ws
}

func (c *Connector) reconnectWS() {
	for {
		log.Printf("[connector] reconnecting WS to %s", c.wsRPC)
		client, err := c.dialWS(c.wsRPC)
		if err != nil {
			log.Printf("[connector] WS reconnect failed: %v — retrying in 2s", err)
			time.Sleep(2 * time.Second)
			continue
		}
		c.mu.Lock()
		c.ws = client
		c.mu.Unlock()
		c.Metrics.Reconnects.Add(1)
		log.Println("[connector] WS reconnected")
		return
	}
}

func (c *Connector) watchLogs(ctx context.Context, query ethereum.FilterQuery, ch chan *types.Log) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		sub, err := c.getWSClient().Subscribe(eth.NewLogs(ch, query))
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
