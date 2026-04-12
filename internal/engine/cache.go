package engine

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// dans engine/cache.go

type SimCache struct {
	mu       sync.Mutex
	failures map[string]int // key = borrower address
}

func NewSimCache() *SimCache {
	return &SimCache{failures: make(map[string]int)}
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
