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
	bestNetOut := big.NewInt(0)

	for _, route := range routes {
		if len(route) == 0 {
			continue
		}

		totalOut := new(big.Int).Set(route[len(route)-1].WCAmountOut) // final output
		totalSlippage := 0.0

		for i := range route {
			if i > 0 {
				// pools gets too many slippage
				if route[i].WCAmountOut.Cmp(route[i-1].WCAmountIn) > 0 {

				}

			}

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
