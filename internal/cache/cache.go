package cache

import (
	"math/big"
	"sync"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/pkg/api"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
)

/*
Cache package holds storage logic
Cache Init
Snapshot

# HF calculation

Pos insert / update / delete logic
*/

type Cache struct {
	Markets   *MarketStore
	MarketMap map[[32]byte]morpho.MarketParams
 ToFlush []BorrowPosition // async flush in sqlite 
 mu sync.RWMutex 
}

type MarketStore struct {
	mu      sync.RWMutex
	markets map[[32]byte]*Market
}


type Market struct {
	Mu          sync.RWMutex
	Canceled    bool
	Oracle      Oracle
	LLTV        *big.Int // already in morphomarket
	Stats       MarketStats
	ActiveIndex int               // index of last pos with tracked HF
	Positions   []*BorrowPosition // Borrow positions sorted by HF asc
}

type Oracle struct {
	Price   *big.Int
	Address common.Address
}

// MaxCollateralPos is used for quoting max slipage
// within api refresh time bigest borrow might be over swappable amount if liquidated fast
type MarketStats struct {
	TotalBorrowAssets, TotalBorrowShares, BorrowRate, MaxCollateralPos *big.Int
	SwapFee                                                            uint32
	LastUpdate                                                         int64
}

type MarketSnapshot struct {
	ID        [32]byte
	Oracle    Oracle
	LLTV      *big.Int
	Stats     MarketStats
	Positions []BorrowPosition
}

func NewStore(markets []morpho.MarketParams) *MarketStore {
	marketsMap := make(map[[32]byte]*Market, len(markets))
	for _, m := range markets {

		market := &Market{
			// might inizialize array
			Positions: make([]*BorrowPosition, 0),
		}
		marketsMap[m.ID] = market
	}

	return &MarketStore{
		mu:      sync.RWMutex{},
		markets: marketsMap,
	}
}

func NewCache(conf lqtypes.Config, markets []morpho.MarketParams, filters api.MarketFilters) *Cache {

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
