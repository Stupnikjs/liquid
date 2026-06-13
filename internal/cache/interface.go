package cache

import (
	"math/big"

	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
)

type MarketCache interface {
	// API
	// Lecture
	Ids() [][32]byte
	GetSnapshot(id [32]byte) *MarketSnapshot
	MorphoMarketByID(id [32]byte) morpho.MarketParams
	AllMarkets() []morpho.MarketParams
	LogMarkets()

	UpdatePositionsSlice(id [32]byte, positions []*BorrowPosition)
	UpdateMarketStats(id [32]byte, stats MarketStats)
	UpdateOnchainRefresh(id [32]byte, totalBorrowShares, totalBorrowAssets, oraclePrice *big.Int)
	UpdateOraclePrice(id [32]byte, oraclePrice *big.Int)
	UpdateMaxCollateralPos(id [32]byte, MaxCollateralPos *big.Int)

	UpdateAccrueInterest(id [32]byte, interest, prevBorrowRate *big.Int)
	UpdateRepay(id [32]byte, onBehalf common.Address, shares *big.Int)
	UpdateBorrow(id [32]byte, onBehalf common.Address, shares *big.Int)
	UpdateSupplyCollateral(id [32]byte, onBehalf common.Address, shares *big.Int)
	UpdateLiquidate(id [32]byte, onBehalf common.Address, repaidShares, badDebtShares *big.Int)

	UpdateRecompute(id [32]byte, tickCount int)
	CancelMarket(id [32]byte)
}
