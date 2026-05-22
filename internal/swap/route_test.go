package swap

import (
	"math"
	"math/big"
	"testing"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
)

func TestEstimateRouteSlippage(t *testing.T) {
	tests := []struct {
		name             string
		route            []lqtypes.PoolEdge
		expectedSlippage float64
		expectedErr      bool
	}{
		{
			name:        "empty route",
			route:       []lqtypes.PoolEdge{},
			expectedErr: true,
		},
		{
			name: "single pool no slippage",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:  big.NewInt(1000),
					WCAmountOut: big.NewInt(997),
					WCSlippage:  0.003, // 0.3% fee
				},
			},
			expectedSlippage: 0.003,
		},
		{
			name: "two pools contigus sans gap de transition",
			// amountOut pool1 == amountIn pool2 → pas de slippage de transition
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:  big.NewInt(1000),
					WCAmountOut: big.NewInt(997),
					WCSlippage:  0.003,
				},
				{
					WCAmountIn:  big.NewInt(997), // exactement ce que pool1 sort
					WCAmountOut: big.NewInt(994),
					WCSlippage:  0.003,
				},
			},
			expectedSlippage: 0.006, // 0.3% + 0.3%, zero transition slippage
		},
		{
			name: "two pools avec gap de transition",
			// pool1 sort 1000 mais pool2 attend 990 → gap de 10
			// transition ratio = 10/1000 = 0.01
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:  big.NewInt(1000),
					WCAmountOut: big.NewInt(1000),
					WCSlippage:  0.003,
				},
				{
					WCAmountIn:  big.NewInt(990),
					WCAmountOut: big.NewInt(987),
					WCSlippage:  0.003,
				},
			},
			// 0.003 + 0.01 (transition) + 0.003 = 0.016
			expectedSlippage: 0.016,
		},
		{
			name: "nil WCAmountIn",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:  nil,
					WCAmountOut: big.NewInt(997),
					WCSlippage:  0.003,
				},
			},
			expectedErr: true,
		},
		{
			name: "zero WCAmountIn",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:  big.NewInt(0),
					WCAmountOut: big.NewInt(997),
					WCSlippage:  0.003,
				},
			},
			expectedErr: true,
		},
		{
			name: "route avec trois pools et deux gaps de transition",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:  big.NewInt(1000),
					WCAmountOut: big.NewInt(1000),
					WCSlippage:  0.003,
				},
				{
					WCAmountIn:  big.NewInt(990), // gap de 10 → 10/1000 = 0.01
					WCAmountOut: big.NewInt(990),
					WCSlippage:  0.003,
				},
				{
					WCAmountIn:  big.NewInt(980), // gap de 10 → 10/990 ≈ 0.01010...
					WCAmountOut: big.NewInt(977),
					WCSlippage:  0.003,
				},
			},
			// 0.003 + 0.01 + 0.003 + 0.01010... + 0.003 = 0.02910...
			expectedSlippage: 0.003 + 0.01 + 0.003 + (10.0 / 990.0) + 0.003,
		},
		{
			name: "pool avec WCAmountOut nul",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:  big.NewInt(1000),
					WCAmountOut: big.NewInt(0),
					WCSlippage:  0.003,
				},
				{
					WCAmountIn:  big.NewInt(990),
					WCAmountOut: big.NewInt(987),
					WCSlippage:  0.003,
				},
			},
			// WCAmountOut == 0 → pas de gap détecté (0 < 990), slippage = WCSlippage uniquement
			// mais semantiquement c'est un pool cassé → selon ta logique métier tu peux vouloir une erreur
			expectedSlippage: 0.006,
		},
		{
			name: "nil WCAmountOut",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:  big.NewInt(1000),
					WCAmountOut: nil,
					WCSlippage:  0.003,
				},
			},
			expectedErr: true,
		},
		{
			name: "nil WCAmountIn sur le deuxième pool",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:  big.NewInt(1000),
					WCAmountOut: big.NewInt(997),
					WCSlippage:  0.003,
				},
				{
					WCAmountIn:  nil,
					WCAmountOut: big.NewInt(994),
					WCSlippage:  0.003,
				},
			},
			expectedErr: true,
		},
		{
			name: "zero WCAmountIn sur le deuxième pool",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:  big.NewInt(1000),
					WCAmountOut: big.NewInt(997),
					WCSlippage:  0.003,
				},
				{
					WCAmountIn:  big.NewInt(0),
					WCAmountOut: big.NewInt(994),
					WCSlippage:  0.003,
				},
			},
			expectedErr: true,
		},
		{
			name: "gap de transition nul exact (amountOut == nextAmountIn)",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:  big.NewInt(1000),
					WCAmountOut: big.NewInt(500),
					WCSlippage:  0.003,
				},
				{
					WCAmountIn:  big.NewInt(500),
					WCAmountOut: big.NewInt(498),
					WCSlippage:  0.003,
				},
			},
			expectedSlippage: 0.006, // pas de gap → slippage de transition = 0
		},
		{
			name: "WCSlippage négatif (données corrompues)",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:  big.NewInt(1000),
					WCAmountOut: big.NewInt(1005), // quoter buggé qui sort plus qu'il reçoit
					WCSlippage:  -0.005,
				},
			},
			// ta fonction additionne sans vérifier → totalSlippage = -0.005
			// c'est un edge case à documenter ou à sanitizer
			expectedSlippage: -0.005,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EstimateRouteSlippage(tt.route)

			if tt.expectedErr {
				if err == nil {
					t.Errorf("attendu une erreur, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("erreur inattendue: %v", err)
			}

			const epsilon = 1e-9
			if math.Abs(got-tt.expectedSlippage) > epsilon {
				t.Errorf("slippage: got %v, want %v", got, tt.expectedSlippage)
			}
		})
	}
}

func TestRouteMaxAmountIn(t *testing.T) {
	e36 := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)

	// prix oracle : 1 collateral = 2 loan → PriceAtQuote = 2e36
	price2x := new(big.Int).Mul(big.NewInt(2), e36)
	// prix oracle : 1 collateral = 1 loan → PriceAtQuote = 1e36
	price1x := new(big.Int).Set(e36)

	tests := []struct {
		name        string
		route       []lqtypes.PoolEdge
		expectedErr bool
		check       func(t *testing.T, result *big.Int)
	}{
		{
			name:        "empty route",
			route:       []lqtypes.PoolEdge{},
			expectedErr: true,
		},
		{
			name: "single hop, pas de slippage",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:   big.NewInt(1000),
					WCAmountOut:  big.NewInt(997),
					WCSlippage:   0.0,
					PriceAtQuote: price1x,
				},
			},
			// bottleneck = hop0, slippage = 0 → result = 1000 / 1.0 = 1000
			check: func(t *testing.T, result *big.Int) {
				if result.Cmp(big.NewInt(1000)) != 0 {
					t.Errorf("got %v, want 1000", result)
				}
			},
		},
		{
			name: "single hop, slippage 2%",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:   big.NewInt(1000),
					WCAmountOut:  big.NewInt(980),
					WCSlippage:   0.02,
					PriceAtQuote: price1x,
				},
			},
			// result = 1000 / (1 - 0.02) = 1000 / 0.98 ≈ 1020
			check: func(t *testing.T, result *big.Int) {
				expected := big.NewInt(1020)
				delta := new(big.Int).Abs(new(big.Int).Sub(result, expected))
				if delta.Cmp(big.NewInt(1)) > 0 {
					t.Errorf("got %v, want ~1020", result)
				}
			},
		},
		{
			name: "deux hops, hop0 est le goulot",
			// hop0: WCAmountIn=500 en token0
			// hop1: WCAmountIn=2000 → converti en token0 = 2000 * e36 / (2*e36) = 1000
			// min = 500 (hop0) → bottleneck=0
			// result = 500 / (1 - 0.01) ≈ 505
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:   big.NewInt(500),
					WCAmountOut:  big.NewInt(495),
					WCSlippage:   0.01,
					PriceAtQuote: price1x,
				},
				{
					WCAmountIn:   big.NewInt(2000),
					WCAmountOut:  big.NewInt(1980),
					WCSlippage:   0.01,
					PriceAtQuote: price2x,
				},
			},
			check: func(t *testing.T, result *big.Int) {
				expected := big.NewInt(505)
				delta := new(big.Int).Abs(new(big.Int).Sub(result, expected))
				if delta.Cmp(big.NewInt(2)) > 0 {
					t.Errorf("got %v, want ~505", result)
				}
			},
		},
		{
			name: "deux hops, hop1 est le goulot",
			// hop0: WCAmountIn=1000 en token0
			// hop1: WCAmountIn=400 → converti = 400 * e36 / (2*e36) = 200
			// min = 200 (hop1) → bottleneck=1
			// result = 200 / (1-0.01) / (1-0.01) ≈ 204
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:   big.NewInt(1000),
					WCAmountOut:  big.NewInt(990),
					WCSlippage:   0.01,
					PriceAtQuote: price1x,
				},
				{
					WCAmountIn:   big.NewInt(400),
					WCAmountOut:  big.NewInt(396),
					WCSlippage:   0.01,
					PriceAtQuote: price2x,
				},
			},
			check: func(t *testing.T, result *big.Int) {
				expected := big.NewInt(204)
				delta := new(big.Int).Abs(new(big.Int).Sub(result, expected))
				if delta.Cmp(big.NewInt(2)) > 0 {
					t.Errorf("got %v, want ~204", result)
				}
			},
		},
		{
			name: "slippage zero sur tous les hops",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:   big.NewInt(1000),
					WCAmountOut:  big.NewInt(1000),
					WCSlippage:   0.0,
					PriceAtQuote: price1x,
				},
				{
					WCAmountIn:   big.NewInt(800),
					WCAmountOut:  big.NewInt(800),
					WCSlippage:   0.0,
					PriceAtQuote: price1x,
				},
			},
			// hop1 converti = 800, min=800, bottleneck=1
			// result = 800 / 1.0 / 1.0 = 800
			check: func(t *testing.T, result *big.Int) {
				if result.Cmp(big.NewInt(800)) != 0 {
					t.Errorf("got %v, want 800", result)
				}
			},
		},
		{
			name: "PriceAtQuote nil sur hop1 → panic guard",
			route: []lqtypes.PoolEdge{
				{
					WCAmountIn:   big.NewInt(1000),
					WCAmountOut:  big.NewInt(990),
					WCSlippage:   0.01,
					PriceAtQuote: price1x,
				},
				{
					WCAmountIn:   big.NewInt(500),
					WCAmountOut:  big.NewInt(495),
					WCSlippage:   0.01,
					PriceAtQuote: nil, // non assigné
				},
			},
			// selon ton guard : skip hop1 → bottleneck=0, result ≈ 1010
			check: func(t *testing.T, result *big.Int) {
				if result == nil {
					t.Error("got nil, want non-nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RouteMaxAmountIn(tt.route)

			if tt.expectedErr {
				if err == nil {
					t.Errorf("attendu une erreur, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("erreur inattendue: %v", err)
			}
			tt.check(t, result)
		})
	}
}
