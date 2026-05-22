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
