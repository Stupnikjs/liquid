package market

import (
	"math/big"
	"sync"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/ethereum/go-ethereum/common"
)

type Market struct {
	Mu        sync.RWMutex
	Canceled  bool
	Oracle    Oracle
	LLTV      *big.Int
	Stats     MarketStats
 Odos      swap.Odos 
	Positions map[common.Address]*BorrowPosition
}

type Oracle struct {
	Price   *big.Int
	Address common.Address
}

type MarketStats struct {
	TotalBorrowAssets, TotalBorrowShares, BorrowRate *big.Int
	LastUpdate                                       int64
}

type MarketSnapshot struct {
	ID        [32]byte
	Oracle    Oracle
	LLTV      *big.Int
	Stats     MarketStats
	Positions []BorrowPosition
}

// AccruedBorrowAssets retourne totalBorrowAssets mis à jour jusqu'à `now`
// To Simulate Morpho call accrue interest at begin of liquidate func
func (ms *MarketStats) AccruedBorrowAssets() *big.Int {
	if ms.TotalBorrowAssets == nil {
		return ms.TotalBorrowAssets
	}
	if ms.TotalBorrowAssets.Sign() == 0 {
		return ms.TotalBorrowAssets
	}

	dt := big.NewInt(time.Now().Unix() - ms.LastUpdate)
	if dt.Sign() <= 0 {
		return ms.TotalBorrowAssets
	}
	if ms.BorrowRate == nil {
		return ms.TotalBorrowAssets
	}
	// interest = totalBorrowAssets * borrowRate * dt / WAD
	interest := new(big.Int).Mul(ms.TotalBorrowAssets, ms.BorrowRate)
	interest.Mul(interest, dt)
	interest.Div(interest, utils.WAD)

	return new(big.Int).Add(ms.TotalBorrowAssets, interest)
}
