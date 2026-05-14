// graph.go
package swap

import (
	"github.com/Stupnikjs/liquid/internal/lqtypes"

	"github.com/ethereum/go-ethereum/common"
)

// Graph : tokenIn -> liste de pools disponibles depuis ce token
type PoolGraph map[common.Address][]lqtypes.PoolEdge

func BuildGraph(pools []lqtypes.PoolEdge) PoolGraph {
	g := make(PoolGraph)
	for _, p := range pools {
		g[p.TokenIn] = append(g[p.TokenIn], p)
	}
	return g
}

func (g PoolGraph) FindRoutes(
	tokenIn, tokenOut common.Address,
	maxHops int,
) [][]lqtypes.PoolEdge {

	var results [][]lqtypes.PoolEdge
	visited := make(map[common.Address]bool)
	path := make([]lqtypes.PoolEdge, 0, maxHops)

	// iteration util found tokenOut
	var dfs func(current common.Address)
	dfs = func(current common.Address) {
		if current == tokenOut {
			cp := make([]lqtypes.PoolEdge, len(path))
			copy(cp, path)
			results = append(results, cp)
			return
		}
		if len(path) >= maxHops {
			return
		}
		visited[current] = true

		for _, edge := range g[current] {
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
