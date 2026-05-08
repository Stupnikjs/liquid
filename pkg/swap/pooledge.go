package swap

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Swaping entity to find best swap in multi hop
type PoolEdge struct {
	TokenIn      common.Address
	TokenOut     common.Address
	Router       common.Address
	Fee          uint32
	WCSlippage   float64
	WCAmountIn   *big.Int
	WCAmountOut  *big.Int
	CalibratedAt time.Time
}

// is older than maxAge ?
func (p PoolEdge) IsStale(maxAge time.Duration) bool {
	return time.Since(p.CalibratedAt) > maxAge
}
