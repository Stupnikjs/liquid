package testutil

import (
	"context"
	"math/big"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
)

type MockCache struct {
	p      *cache.BorrowPosition
	oracle cache.Oracle
	LLTV   *big.Int
	Stats  *cache.MarketStats
}

func (m *MockCache) ApiCall() error                       { return nil }
func (m *MockCache) ApiResyncRoutine(ctx context.Context) {}
func (m *MockCache) Ids() [][32]byte {
	hexStr := "8793cf302b8ffd655ab97bd1c695dbd967807e8367a65cb2f4edaf1380ba1bda"
	b := common.Hex2Bytes(hexStr)
	var id [32]byte
	copy(id[:], b)
	return [][32]byte{id}
}
func (m *MockCache) GetSnapshot(id [32]byte) *cache.MarketSnapshot {
	return &cache.MarketSnapshot{
		ID:        m.p.MarketID,
		Oracle:    m.oracle,
		LLTV:      m.LLTV,
		Stats:     *m.Stats,
		Positions: []cache.BorrowPosition{*m.p},
	}
}
func (m *MockCache) MarketByID(id [32]byte) morpho.MarketParams {
	return morpho.MarketParams{
		LoanToken:               usdc,
		CollateralToken:         weth,
		Oracle:                  common.HexToAddress("0xFEa2D58cEfCb9fcb597723c6bAE66fFE4193aFE4"),
		Irm:                     common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687"),
		LLTV:                    big.NewInt(860000000000000000),
		ChainID:                 8453,
		LoanTokenStr:            "USDC",
		CollateralTokenStr:      "WETH",
		LoanTokenDecimals:       6,
		CollateralTokenDecimals: 18,
	}
}

func AllMarkets() []morpho.MarketParams {
	return []morpho.MarketParams{
		{
			LoanToken:               usdc,
			CollateralToken:         weth,
			Oracle:                  common.HexToAddress("0xFEa2D58cEfCb9fcb597723c6bAE66fFE4193aFE4"),
			Irm:                     common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687"),
			LLTV:                    big.NewInt(860000000000000000),
			ChainID:                 8453,
			LoanTokenStr:            "USDC",
			CollateralTokenStr:      "WETH",
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
		},
	}
}
func LogMarkets()

func UpdatePositionsSlice(id [32]byte, positions []*cache.BorrowPosition)                          {}
func UpdateMarketStats(id [32]byte, stats cache.MarketStats)                                       {}
func UpdateOnchainRefresh(id [32]byte, totalBorrowShares, totalBorrowAssets, oraclePrice *big.Int) {}
func UpdateOraclePrice(id [32]byte, oraclePrice *big.Int)                                          {}
func UpdateMaxCollateralPos(id [32]byte, MaxCollateralPos *big.Int)                                {}

func UpdateAccrueInterest(id [32]byte, interest, prevBorrowRate *big.Int)                        {}
func UpdateRepay(id [32]byte, onBehalf common.Address, shares *big.Int)                          {}
func UpdateBorrow(id [32]byte, onBehalf common.Address, shares *big.Int)                         {}
func UpdateSupplyCollateral(id [32]byte, onBehalf common.Address, shares *big.Int)               {}
func UpdateLiquidate(id [32]byte, onBehalf common.Address, repaidShares, badDebtShares *big.Int) {}

func UpdateRecompute(id [32]byte, tickCount int) {}
func CancelMarket(id [32]byte)                   {}
