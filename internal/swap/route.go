package swap

import (
	"math/big"
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
	Mu     sync.RWMutex
	Routes map[RouteKey]Route
}

type Route struct {
	Hops        []lqtypes.PoolEdge // ou Legs, Steps
	WCAmountOut *big.Int           // worst case du chemin entier, précalculé
}

func NewRouteCache(n int) *RouteCache {
	return &RouteCache{
		Routes: make(map[RouteKey]Route, n),
	}
}

func (rc *RouteCache) StoreRoute(r []lqtypes.PoolEdge) {
	if len(r) == 0 {
		return
	}
	lastWC := r[len(r)-1].WCAmountOut
	if lastWC == nil {
		return // ou log + return selon ta logique
	}

	k := RouteKey{
		TokenIn:  r[0].TokenIn,
		TokenOut: r[len(r)-1].TokenOut,
	}

	rc.Mu.Lock()
	defer rc.Mu.Unlock()
	hops := make([]lqtypes.PoolEdge, len(r))
	copy(hops, r)
	route := Route{
		Hops:        hops,
		WCAmountOut: new(big.Int).Set(lastWC), // copie aussi le big.Int
	}
	rc.Routes[k] = route
}

func (rc *RouteCache) GetRoute(tokenIn, tokenOut common.Address) (Route, bool) {
	k := RouteKey{
		TokenIn:  tokenIn,
		TokenOut: tokenOut,
	}
	rc.Mu.RLock()
	defer rc.Mu.RUnlock()
	route, ok := rc.Routes[k]
	if !ok {
		return Route{}, false
	}
	hops := make([]lqtypes.PoolEdge, len(route.Hops))
	copy(hops, route.Hops)

	return Route{
		Hops:        hops,
		WCAmountOut: new(big.Int).Set(route.WCAmountOut),
	}, true
}
