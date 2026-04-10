package engine

import (
	"math/big"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
)

type SimResult struct {
	Position     market.BorrowPosition
	MarketID     [32]byte
	RepayShares  *big.Int
	SeizeAssets  *big.Int
	GasEstimate  uint64
	EstProfit    *big.Int
	IsLiquidable bool
	SimErr       error
}
