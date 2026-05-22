package swap

import (
	"fmt"
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

// BestRoute returns the route with the highest expected output after slippage + basic sanity checks

func BestRoute(routes [][]lqtypes.PoolEdge, minProfit *big.Int) ([]lqtypes.PoolEdge, error) {
	if len(routes) == 0 {
		return nil, fmt.Errorf("no routes provided")
	}

	var bestRoute []lqtypes.PoolEdge
	var lowestSlippage float64 = 1.0

	for _, route := range routes {

		slippage, err := EstimateRouteSlippage(route)

		if err != nil {
			fmt.Printf("Error estimating slippage for route: %v\n", err)
			continue
		}
		if slippage < lowestSlippage {
			bestRoute = route
		}
	}

	return bestRoute, nil
}

func EstimateRouteSlippage(route []lqtypes.PoolEdge) (float64, error) {
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
