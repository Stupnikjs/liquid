// graph.go
package swap

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Graph : tokenIn -> liste de pools disponibles depuis ce token
type PoolGraph map[common.Address][]PoolEdge

func (rc *RouteCache) PoolsToGraph() {
	rc.Mu.Lock()
	defer rc.Mu.Unlock()
	rc.Graph = make(PoolGraph)
	for _, p := range rc.Pools {
		rc.Graph[p.TokenIn] = append(rc.Graph[p.TokenIn], p)
	}
}

func (rc *RouteCache) FindRoutes(
	tokenIn, tokenOut common.Address,
	maxHops int,
) [][]PoolEdge {

	rc.Mu.RLock()
	defer rc.Mu.RUnlock()
	var results [][]PoolEdge
	visited := make(map[common.Address]bool)
	path := make([]PoolEdge, 0, maxHops)

	// iteration util found tokenOut
	var dfs func(current common.Address)
	dfs = func(current common.Address) {
		if current == tokenOut {
			cp := make([]PoolEdge, len(path))
			copy(cp, path)
			results = append(results, cp)
			return
		}
		if len(path) >= maxHops {
			return
		}
		visited[current] = true

		for _, edge := range rc.Graph[current] {
			if !visited[edge.TokenOut] {
				// appening token to path
				path = append(path, edge)
				// recursive call
				dfs(edge.TokenOut)

				path = path[:len(path)-1]
			}
		}
		visited[current] = false
	}

	dfs(tokenIn)
	return results
}

func (rc *RouteCache) FindBestRoute(tokenIn, tokenOut common.Address) ([]PoolEdge, bool) {
	routes := rc.FindRoutes(tokenIn, tokenOut, MAX_HOP_LEN) // Assuming maxHops is 3
	if len(routes) == 0 {
		return nil, false
	}
	bestRoute, err := BestRoute(routes, big.NewInt(0))
	if err != nil {
		return nil, false
	}
	return bestRoute, true
}
