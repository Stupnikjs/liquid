package swap

import (
	"github.com/ethereum/go-ethereum/common"
)

// swap  in:token address out:token address slippage
// double swap slippage_final = s1 + s2 - s1 * s2

type SwapManager struct {
	Graph *PoolGraph
}

func NewSwapManager() *SwapManager {
	return &SwapManager{Graph: NewPoolGraph()}
}

// LiquidityGraph agrège tous les pools connus
type PoolGraph struct {
	// token → liste de pools partant de ce token
	Edges map[common.Address][]PoolEdge
}

func NewPoolGraph() *PoolGraph {
	return &PoolGraph{Edges: make(map[common.Address][]PoolEdge)}
}

func (g *PoolGraph) AddPool(edge PoolEdge) {
	g.Edges[edge.TokenIn] = append(g.Edges[edge.TokenIn], edge)
}

// FindRoutes retourne toutes les routes possibles de tokenIn à tokenOut
// avec au maximum maxHops sauts
func (g *PoolGraph) FindRoutes(
	tokenIn, tokenOut common.Address,
	maxHops int,
) [][]PoolEdge {
	var results [][]PoolEdge
	visited := make(map[common.Address]bool)

	var dfs func(current common.Address, path []PoolEdge)
	dfs = func(current common.Address, path []PoolEdge) {
		if len(path) > maxHops {
			return
		}

		if current == tokenOut && len(path) > 0 {
			route := make([]PoolEdge, len(path))
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
