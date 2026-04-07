package core

import (
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/pkg/cex"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
)

type CacheConfig struct {
	Markets map[[32]byte]morpho.MarketParams
	Chain   morpho.ChainConfig
}

type RebuildEvent struct {
	MarketIDs [][32]byte // which markets were refreshed
	Reason    string     // "onchain_refresh", "price_update", etc.
}

type Cache struct {
	Config           CacheConfig
	CexCache         *cex.CexCache
	PositionCache    *PositionCache
	EthCallCount     atomic.Int64
	LastMinCallCount atomic.Int64
	watchlist        []*Liquidable // trié par HF asc
	watchMu          sync.RWMutex
	rebuildCh        chan RebuildEvent
	liquidCh         chan Liquidable
}

type PositionCache struct {
	m map[[32]byte]*Market
}

type Market struct {
	Mu sync.RWMutex
	MarketCache
	MarketStats
}

type MarketStats struct {
	OraclePrice, TotalBorrowAssets, TotalBorrowShares, LLTV, BorrowRate *big.Int
	LastUpdate                                                          int64
}

type MarketCache struct {
	Oracle    common.Address
	Positions map[common.Address]*BorrowPosition
}

type Liquidable struct {
	Pos          *BorrowPosition
	MarketID     [32]byte
	HF           *big.Int
	RepayShares  *big.Int
	SeizeAssets  *big.Int
	EstProfit    *big.Int
	GasEstimate  uint64
	SimulatedAt  time.Time
	SimErr       error
	IsLiquidable bool
}
