package swap

import (
	"github.com/ethereum/go-ethereum/common"
)

// swap  in:token address out:token address slippage
// double swap slippage_final = s1 + s2 - s1 * s2

type SwapManager struct {
	Graph *LiquidityGraph
}

func NewSwapManager() *SwapManager {
	return &SwapManager{Graph: NewLiquidityGraph()}
}

/*
	type PoolEdge struct {
		TokenIn  common.Address
		TokenOut common.Address
		Router   common.Address
		Fee      uint32

		// Métrique de liquidité — calibrée avec un montant "worst case"
		WCSlippage   float64   // slippage observé pour WCAmountIn
		WCAmountIn   *big.Int  // montant utilisé pour calibrer (ex: position max du marché)
		CalibratedAt time.Time // pour savoir si le cache est périmé
	}
*/

// LiquidityGraph agrège tous les pools connus
type LiquidityGraph struct {
	// token → liste de pools partant de ce token
	Edges map[common.Address][]PoolEdge
}

func NewLiquidityGraph() *LiquidityGraph {
	return &LiquidityGraph{Edges: make(map[common.Address][]PoolEdge)}
}

func (g *LiquidityGraph) AddPool(edge PoolEdge) {
	g.Edges[edge.TokenIn] = append(g.Edges[edge.TokenIn], edge)
}

// FindRoutes retourne toutes les routes possibles de tokenIn à tokenOut
// avec au maximum maxHops sauts
func (g *LiquidityGraph) FindRoutes(
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
