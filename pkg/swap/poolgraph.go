package swap

import (
	"github.com/Stupnikjs/liquid/pkg/types"
	"github.com/ethereum/go-ethereum/common"
)

type SwapManager struct {
	Graph *PoolGraph
}

func NewSwapManager() *SwapManager {
	return &SwapManager{Graph: NewPoolGraph()}
}

// LiquidityGraph agrège tous les pools connus
type PoolGraph struct {
	// token → liste de pools partant de ce token
	Edges map[common.Address][]types.PoolEdge
}

func NewPoolGraph() *PoolGraph {
	return &PoolGraph{Edges: make(map[common.Address][]types.PoolEdge)}
}

func (g *PoolGraph) AddPool(edge types.PoolEdge) {
	g.Edges[edge.TokenIn] = append(g.Edges[edge.TokenIn], edge)
}

// FindRoutes retourne toutes les routes possibles de tokenIn à tokenOut
// avec au maximum maxHops sauts
func (g *PoolGraph) FindRoutes(
	tokenIn, tokenOut common.Address,
	maxHops int,
) [][]types.PoolEdge {
	var results [][]types.PoolEdge
	visited := make(map[common.Address]bool)

	var dfs func(current common.Address, path []types.PoolEdge)
	dfs = func(current common.Address, path []types.PoolEdge) {
		if len(path) > maxHops {
			return
		}

		if current == tokenOut && len(path) > 0 {
			route := make([]types.PoolEdge, len(path))
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
