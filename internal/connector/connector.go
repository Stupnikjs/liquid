package connector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

type Connector struct {
	WS         []string
	HTTP       []string
	MainIndex  int
	ethCalls   atomic.Uint64
	mu         sync.RWMutex
	currHTTP   int
	currWS     int
	ClientHTTP *w3.Client
	ClientWS   *w3.Client
	PositionCh chan *types.Log
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

func (conn *Connector) LogsEthCallsFromLastMin(ctx context.Context, logChan chan string) {
	utils.RunTicker(ctx, time.Minute, func() {
		count := conn.ethCalls.Load()
		logChan <- fmt.Sprintf("%d ETH_CALLS \n", count)
		conn.ethCalls.Store(0)
	})

}

// func (c *w3.Client) CallCtx(ctx context.Context, calls ...w3types.RPCCaller) error
func (conn *Connector) EthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error {
	defer conn.ethCalls.Add(uint64(len(calls)))
	return conn.ClientHTTP.CallCtx(ctx, calls...)
}
