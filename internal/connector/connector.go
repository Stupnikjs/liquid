package connector

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"golang.org/x/time/rate"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

type Connector struct {
	ethCalls           atomic.Uint64
	mu                 sync.RWMutex
	MainRPC            string
	QuoteRPC           string
	WsRPC              string
	ClientHTTP         *w3.Client
	ClientHTTPFallback *w3.Client
	ClientWS           *w3.Client
	PositionCh         chan *types.Log // usefull ?
	limiter            *rate.Limiter
}

func NewConnector(mainRPC, quoteRPC, wsRPC string) *Connector {
	clientHTTP, err := w3.Dial(mainRPC)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	clientHTTPFallback, err := w3.Dial(quoteRPC)
	if err != nil {
		fmt.Println("fallback dial failed:", err)
		panic(err)
	}
	clientWS, err := w3.Dial(wsRPC)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	return &Connector{
		MainRPC:            mainRPC,
		QuoteRPC:           quoteRPC,
		WsRPC:              wsRPC,
		ClientHTTP:         clientHTTP,
		ClientHTTPFallback: clientHTTPFallback,
		ClientWS:           clientWS,
		PositionCh:         make(chan *types.Log, 100),
		limiter:            rate.NewLimiter(rate.Every(time.Minute/300), 10),
	}
}

func (c *Connector) getWSClient() *w3.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ClientWS
}

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

func (conn *Connector) LogsEthCallsNum(ctx context.Context, logChan chan string) {
	utils.RunTicker(ctx, 10*time.Minute, func() {
		count := conn.ethCalls.Load()
		logChan <- fmt.Sprintf("%d ETH_CALLS  \n", count)
		conn.ethCalls.Store(0)
	})

}

// func (c *w3.Client) CallCtx(ctx context.Context, calls ...w3types.RPCCaller) error
func (conn *Connector) EthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error {
	if err := conn.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit: %w", err)
	}
	conn.mu.RLock()
	client := conn.ClientHTTP
	conn.mu.RUnlock()
	defer conn.ethCalls.Add(uint64(len(calls)))
	err := client.CallCtx(ctx, calls...)
	if err != nil && strings.Contains(err.Error(), "Request timeout") {
		return conn.ClientHTTPFallback.CallCtx(ctx, calls...)
	}
	return err
}
