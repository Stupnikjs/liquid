package swap

import (
	"sync"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/ethereum/go-ethereum/common"
)

type SwapManager struct {
	Graph *PoolGraph
	Cache RouteCache
}

func NewSwapManager() *SwapManager {
	return &SwapManager{Graph: NewPoolGraph()}
}

// LiquidityGraph agrège tous les pools connus
type PoolGraph struct {
	// token → liste de pools partant de ce token
	Edges map[common.Address][]lqtypes.PoolEdge
}

func NewPoolGraph() *PoolGraph {
	return &PoolGraph{Edges: make(map[common.Address][]lqtypes.PoolEdge)}
}

func (g *PoolGraph) AddPool(edge lqtypes.PoolEdge) {
	g.Edges[edge.TokenIn] = append(g.Edges[edge.TokenIn], edge)
}

// FindRoutes retourne toutes les routes possibles de tokenIn à tokenOut
// avec au maximum maxHops sauts
func (g *PoolGraph) FindRoutes(
	tokenIn, tokenOut common.Address,
	maxHops int,
) [][]lqtypes.PoolEdge {
	var results [][]lqtypes.PoolEdge
	visited := make(map[common.Address]bool)

	var dfs func(current common.Address, path []lqtypes.PoolEdge)
	dfs = func(current common.Address, path []lqtypes.PoolEdge) {
		if len(path) > maxHops {
			return
		}

		if current == tokenOut && len(path) > 0 {
			route := make([]lqtypes.PoolEdge, len(path))
			copy(route, path)
			results = append(results, route)
			return
		}
		visited[current] = true
		for _, edge := range g.Edges[current] {
			if !visited[edge.TokenOut] {
				dfs(edge.TokenOut, append(path, edge))
			}
		}
		visited[current] = false // backtrack
	}

	dfs(tokenIn, nil)
	return results
}

func (sm *SwapManager) BestRoute(tokenIn, tokenOut common.Address) ([]lqtypes.PoolEdge, bool) {
	sm.Cache.Mu.RLock()
	defer sm.Cache.Mu.RUnlock()

	r, ok := sm.Cache.Routes[RouteKey{
		TokenIn:  tokenIn,
		TokenOut: tokenOut,
	}]
	return r, ok
}

type RouteKey struct {
	TokenIn  common.Address
	TokenOut common.Address
}

type RouteCache struct {
	Mu     sync.RWMutex
	Routes map[RouteKey][]lqtypes.PoolEdge
}

func (rc *RouteCache) StoreRoute(r []lqtypes.PoolEdge) {
	if len(r) == 0 {
		return
	}
	k := RouteKey{
		TokenIn:  r[0].TokenIn,
		TokenOut: r[len(r)-1].TokenOut,
	}
	rc.Mu.Lock()
	defer rc.Mu.Unlock()
	rc.Routes[k] = r
}
