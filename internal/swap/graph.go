// graph.go
package swap

import (
	"fmt"
	"math/big"

	"github.com/Stupnikjs/liquid/internal/lqtypes"

	"github.com/ethereum/go-ethereum/common"
)

// Graph : tokenIn -> liste de pools disponibles depuis ce token
type PoolGraph map[common.Address][]lqtypes.PoolEdge

func (rc *RouteCache) PoolsToGraph() {
	rc.Mu.Lock()
	defer rc.Mu.Unlock()
	rc.Graph = make(PoolGraph)
	for _, p := range rc.Pools {
		rc.Graph[p.TokenIn] = append(rc.Graph[p.TokenIn], p)
	}
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

// BestRoute returns the route with the highest expected output after slippage + basic sanity checks

func BestRoute(routes [][]lqtypes.PoolEdge, minProfit *big.Int) ([]lqtypes.PoolEdge, error) {
	if len(routes) == 0 {
		return nil, fmt.Errorf("no routes provided")
	}

	var bestRoute []lqtypes.PoolEdge
	bestNetOut := big.NewInt(0)

	for _, route := range routes {
		if len(route) == 0 {
			continue
		}

		totalOut := new(big.Int).Set(route[len(route)-1].WCAmountOut) // final output
		totalSlippage := 0.0

		for _, edge := range route {
			totalSlippage += edge.WCSlippage
		}

		// Apply cumulative slippage
		netOut := applySlippage(totalOut, totalSlippage)

		// Optional: penalize longer routes (extra gas + more risk)
		routePenalty := float64(len(route)-1) * 0.002 // 0.2% penalty per extra hop
		netOut = applySlippage(netOut, routePenalty)

		if netOut.Cmp(bestNetOut) > 0 && netOut.Cmp(minProfit) >= 0 {
			bestNetOut = netOut
			bestRoute = route
		}
	}

	if len(bestRoute) == 0 {
		return nil, fmt.Errorf("no profitable route found")
	}

	return bestRoute, nil
}

// Helper: amount * (1 - slippage)
func applySlippage(amount *big.Int, slippage float64) *big.Int {
	if slippage <= 0 {
		return new(big.Int).Set(amount)
	}

	factor := big.NewFloat(1 - slippage)
	result := new(big.Float).SetInt(amount)
	result = result.Mul(result, factor)

	out := new(big.Int)
	result.Int(out)
	return out
}

func (g PoolGraph) FindBestRoute(tokenIn, tokenOut common.Address) ([]lqtypes.PoolEdge, bool) {
	routes := g.FindRoutes(tokenIn, tokenOut, 3) // Assuming maxHops is 3
	if len(routes) == 0 {
		return nil, false
	}
	bestRoute, err := BestRoute(routes, big.NewInt(0))
	if err != nil {
		return nil, false
	}
	return bestRoute, true
}
