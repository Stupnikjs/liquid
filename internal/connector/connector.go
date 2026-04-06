package connector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

type Connector struct {
	WS        []string
	HTTP      []string
	MainIndex int

	mu         sync.RWMutex
	currHTTP   int
	currWS     int
	ClientHTTP *w3.Client
	ClientWS   *w3.Client

	PositionCh chan *types.Log
	OracleCh   chan *types.Log
}

func NewConnector(httpRPC, websocket []string) *Connector {
	clientHTTP, err := w3.Dial(httpRPC[1])
	if err != nil {
		panic(err)
	}
	clientWS, err := w3.Dial(websocket[1])
	if err != nil {
		panic(err)
	}
	return &Connector{
		WS:         websocket,
		currHTTP:   1,
		currWS:     1,
		HTTP:       httpRPC,
		ClientHTTP: clientHTTP,
		ClientWS:   clientWS,
		OracleCh:   make(chan *types.Log, 100),
		PositionCh: make(chan *types.Log, 100),
	}
}

func (c *Connector) getWSClient() *w3.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ClientWS
}

func (c *Connector) WatchPositions(ctx context.Context) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{config.MorphoMain},
		Topics: [][]common.Hash{{
			config.EventBorrow.Topic0,
			config.EventRepay.Topic0,
			config.EventSupplyCollateral.Topic0,
			config.EventLiquidate.Topic0,
			config.EventAccrueInterest.Topic0,
		}},
	}
	c.watchLogs(ctx, query, c.PositionCh)
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
			log.Printf("[connector] sub error: %v — reconnecting", err)
			sub.Unsubscribe()
			c.reconnectWS()
		case <-ctx.Done():
			sub.Unsubscribe()
			return
		}
	}
}

func (c *Connector) reconnectWS() {
	backoff := 1 * time.Second
	for i := range c.WS {
		endpoint := c.WS[(c.currWS+i)%len(c.WS)]
		client, err := w3.Dial(endpoint)
		if err != nil {
			fmt.Printf("ws reconnect failed for %s: %v, retry in %s\n", endpoint, err, backoff)
			time.Sleep(backoff)
			backoff = min(backoff*2, 30*time.Second)
			continue
		}

		c.mu.Lock()
		old := c.ClientWS
		c.ClientWS = client
		c.currWS = (c.currWS + i) % len(c.WS)
		c.mu.Unlock()

		if old != nil {
			old.Close()
		}
		log.Printf("[connector] WS reconnected to %s", endpoint)
		return
	}
	// All endpoints failed — keep retrying from start after delay
	time.Sleep(backoff)
	c.reconnectWS()
}

// SwapToMainHttp tries the main index first, falls back to rotating through others.
func (c *Connector) SwapToMainHttp() (bool, error) {
	client, err := w3.Dial(c.HTTP[c.MainIndex])

	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		c.currHTTP = (c.currHTTP + 1) % len(c.HTTP)
		fallback, err2 := w3.Dial(c.HTTP[c.currHTTP])
		if err2 != nil {
			return false, fmt.Errorf("swap to main failed, fallback also failed: %w", err2)
		}
		if c.ClientHTTP != nil {
			c.ClientHTTP.Close()
		}
		c.ClientHTTP = fallback
		return false, fmt.Errorf("swap to main failed, using fallback: %w", err)
	}

	if c.ClientHTTP != nil {
		c.ClientHTTP.Close()
	}
	c.ClientHTTP = client
	c.currHTTP = c.MainIndex
	return true, nil
}

func (c *Connector) RefreshRPC() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.currHTTP = (c.currHTTP + 1) % len(c.HTTP)
	client, err := w3.Dial(c.HTTP[c.currHTTP])
	if err != nil {
		return fmt.Errorf("RefreshRPC: failed to dial %s: %w", c.HTTP[c.currHTTP], err)
	}
	if c.ClientHTTP != nil {
		c.ClientHTTP.Close()
	}
	c.ClientHTTP = client
	return nil
}
