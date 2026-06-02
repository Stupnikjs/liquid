package swap

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

var MAX_HOP_LEN = 2

type RouteKey struct {
	TokenIn  common.Address
	TokenOut common.Address
}

type RouteCache struct {
	Mu    sync.RWMutex
	Pools []PoolEdge
	Graph PoolGraph
}

func NewRouteCache() *RouteCache {
	return &RouteCache{
		Pools: make([]PoolEdge, 0),
		Graph: make(PoolGraph),
	}
}
func (rc *RouteCache) AppendPool(r PoolEdge) {
	rc.Mu.Lock()
	defer rc.Mu.Unlock()
	rc.Pools = append(rc.Pools, r)
}

// BestRoute returns the route with the highest expected output after slippage + basic sanity checks

func BestRoute(routes [][]PoolEdge, minProfit *big.Int) ([]PoolEdge, error) {
	if len(routes) == 0 {
		return nil, fmt.Errorf("no routes provided")
	}

	var bestRoute []PoolEdge
	var lowestSlippage float64 = 1.0

	for _, route := range routes {

		slippage, err := EstimateRouteSlippage(route)

		if err != nil {
			fmt.Printf("Error estimating slippage for route: %v\n", err)
			continue
		}
		if slippage < lowestSlippage {
			bestRoute = route
			lowestSlippage = slippage
		}
	}
	if bestRoute == nil {
		return nil, fmt.Errorf("no valid route found")
	}
	return bestRoute, nil
}

func EstimateRouteSlippage(route []PoolEdge) (float64, error) {
	if len(route) == 0 {
		return 0, fmt.Errorf("empty route")
	}

	totalSlippage := 0.0

	for i, pool := range route {
		if pool.WCAmountIn == nil || pool.WCAmountOut == nil || pool.WCAmountIn.Sign() == 0 {
			return 0, fmt.Errorf("hop %d: nil or zero WCAmountIn", i)
		}

		// 1. Slippage interne au pool (fee + price impact)
		totalSlippage += pool.WCSlippage

		// 2. Slippage de transition vers le pool suivant
		if i < len(route)-1 {
			nextPool := route[i+1]
			if nextPool.WCAmountIn == nil || nextPool.WCAmountIn.Sign() == 0 {
				return 0, fmt.Errorf("hop %d: nil or zero WCAmountIn on next pool", i+1)
			}

			if pool.WCAmountOut.Cmp(nextPool.WCAmountIn) > 0 {
				diff := new(big.Int).Sub(pool.WCAmountOut, nextPool.WCAmountIn)

				diffF := new(big.Float).SetInt(diff)
				outF := new(big.Float).SetInt(pool.WCAmountOut)

				ratio, _ := new(big.Float).Quo(diffF, outF).Float64()
				totalSlippage += ratio
			}
		}
	}

	return totalSlippage, nil
}

func RouteMaxAmountIn(route []PoolEdge) (*big.Int, error) {
	if len(route) == 0 {
		return nil, fmt.Errorf("empty route")
	}

	e36 := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)

	// 1. Convertir tous les WCAmountIn en token[0] et trouver le minimum
	bottleneckHop := 0
	minInToken0 := new(big.Int).Set(route[0].WCAmountIn)

	for i := 1; i < len(route); i++ {
		if route[i].PriceAtQuote == nil || route[i].PriceAtQuote.Sign() == 0 {
			continue
		}
		// WCAmountIn[i] en token[0] = WCAmountIn[i] * e36 / PriceAtQuote
		converted := new(big.Int).Mul(route[i].WCAmountIn, e36)
		converted.Div(converted, route[i].PriceAtQuote)

		if converted.Cmp(minInToken0) < 0 {
			minInToken0 = converted
			bottleneckHop = i
		}
	}

	// 2. Restituer les slippages successifs jusqu'au hop goulot
	result := new(big.Float).SetInt(minInToken0)
	for i := bottleneckHop; i >= 0; i-- {
		// amountIn = amountOut / (1 - slippage)
		slippageMul := new(big.Float).SetFloat64(1 - route[i].WCSlippage)
		result.Quo(result, slippageMul)
	}

	out, _ := result.Int(nil)
	return out, nil

}
