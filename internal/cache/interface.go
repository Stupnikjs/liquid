package cache

import (
	"math/big"

	"github.com/Stupnikjs/liquid/pkg/morpho"
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
	CancelMarket(id [32]byte)

	/*
		MarketRoutine(ctx context.Context, liquidationCh chan cache.BorrowPosition, id [32]byte)
		MarketInitTicker(ctx context.Context, id [32]byte) (*time.Ticker, time.Duration)
		MarketTick(ctx context.Context, ms *marketState, id [32]byte, liquidationCh chan cache.BorrowPosition) time.Duration
		MarketOnchainRefresh(ctx context.Context, ms *marketState, id [32]byte)
		MarketRecompute(ms *marketState, id [32]byte)
	*/
}
