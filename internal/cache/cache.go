package cache

import (
	"math/big"
	"sync"

	"github.com/Stupnikjs/liquid/internal/connector"
	"github.com/Stupnikjs/liquid/pkg/api"
	"github.com/Stupnikjs/liquid/pkg/config"
	"github.com/Stupnikjs/liquid/pkg/morpho"
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
	Mu          sync.RWMutex
	Canceled    bool
	Oracle      Oracle
	LLTV        *big.Int
	Stats       MarketStats
	ActiveIndex int               // index of last pos with tracked HF
	Positions   []*BorrowPosition // Borrow positions sorted by HF asc
}

type MarketReader interface {
	Ids() [][32]byte
	GetSnapshot(id [32]byte) *MarketSnapshot
	Update(id [32]byte, fn func(m *Market))
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
	result, err := api.QueryMarkets(conn, conf.ChainID)
	if err != nil {
		return nil
	}

	markets := api.FilterMarket(result, filters, conf.ChainID)
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
