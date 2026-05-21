package swap

import (
	"sync"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/ethereum/go-ethereum/common"
)

var MAX_HOP_LEN = 2

type RouteKey struct {
	TokenIn  common.Address
	TokenOut common.Address
}

type RouteCache struct {
	Mu    sync.RWMutex
	Pools []lqtypes.PoolEdge
	Graph PoolGraph
}

func NewRouteCache() *RouteCache {
	return &RouteCache{
		Pools: make([]lqtypes.PoolEdge, 0),
		Graph: make(PoolGraph),
	}
}
func (rc *RouteCache) AppendPool(r lqtypes.PoolEdge) {
	rc.Mu.Lock()
	defer rc.Mu.Unlock()
	rc.Pools = append(rc.Pools, r)
}
