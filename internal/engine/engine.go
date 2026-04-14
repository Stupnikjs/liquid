package engine

import (
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
)

type Engine struct {
	conn        *connector.Connector
	conf        config.Config
	marketMap   map[[32]byte]morpho.MarketParams
	simCache    *SimCache
	logChan     chan string
	RebuildCh   chan bool
	LiquidateCh chan *Liquidable
}

type SimCache struct {
	mu       sync.Mutex
	failures map[string]int // key = borrower address
}

func NewSimCache() *SimCache {
	return &SimCache{failures: make(map[string]int)}
}

func NewEngine(conn *connector.Connector, conf config.Config, logger chan string) *Engine {
	return &Engine{
		conn:        conn,
		conf:        conf,
		logChan:     logger,
		RebuildCh:   make(chan bool, 1),
		LiquidateCh: make(chan *Liquidable, 1),
	}
}

func (c *SimCache) Blacklisted(addr common.Address) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failures[addr.Hex()] >= 20
}

func (c *SimCache) RecordFailure(addr common.Address) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures[addr.Hex()]++
}

func (c *SimCache) Reset(addr common.Address) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.failures, addr.Hex())
}
