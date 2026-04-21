package cache

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
)

type Cache struct {
	Markets   *MarketStore
	MarketMap map[[32]byte]morpho.MarketParams
}

type MarketStore struct {
	mu      sync.RWMutex
	markets map[[32]byte]*Market
}

type Market struct {
	Mu        sync.RWMutex
	Canceled  bool
	Oracle    Oracle
	LLTV      *big.Int
	Stats     MarketStats
	Positions []*BorrowPosition // Borrow positions sorted by HF asc
}

type Oracle struct {
	Price   *big.Int
	Address common.Address
}

type MarketStats struct {
	TotalBorrowAssets, TotalBorrowShares, BorrowRate, MaxCollateralPos, MaxUniSwappable *big.Int
	SwapFee                                                                             uint32
	LastUpdate                                                                          int64
}

type MarketSnapshot struct {
	ID        [32]byte
	Oracle    Oracle
	LLTV      *big.Int
	Stats     MarketStats
	Positions []BorrowPosition
}

func NewCache(conn *connector.Connector, conf config.Config, filters api.MarketFilters) *Cache {
	result, err := api.QueryMarkets(conn.ClientHTTP, conf.ChainID)
	if err != nil {
		return nil
	}

	markets := api.FilterMarket(result, filters, conf.ChainID)
	fmt.Println("here", len(markets))
	marketMap := make(map[[32]byte]morpho.MarketParams, len(markets))
	store := NewStore(markets)
	for _, mk := range markets {
		marketMap[mk.ID] = mk
		store.Update(mk.ID, func(m *Market) {
			m.LLTV = mk.LLTV
			m.Oracle.Address = mk.Oracle
		})
	}

	return &Cache{
		Markets:   store,
		MarketMap: marketMap, // immutable
	}
}

func (c *Cache) GetMorphoMarketFromId(id [32]byte) morpho.MarketParams {
	return c.MarketMap[id]
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
