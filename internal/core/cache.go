package core

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/pkg/cex"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/w3types"
)

/*

ChainParams: Adresse du contrat/ du swap / RPC
Faire tourner les deux en même temps
Calculer la latence
*/

func NewCache(config CacheConfig) *Cache {

	return &Cache{
		Config:        config,
		CexCache:      cex.NewCexCache(),
		PositionCache: NewPositionCache(config.Markets),
		EthCallCount:  atomic.Int64{},
		rebuildCh:     make(chan RebuildEvent, 1),
	}
}

func NewCacheConfig(markets []morpho.MarketParams, chainConfig morpho.ChainConfig) CacheConfig {
	marketMap := make(map[[32]byte]morpho.MarketParams, len(markets))
	for _, m := range markets {
		marketMap[m.ID] = m
	}
	return CacheConfig{
		Markets: marketMap,
		Chain:   chainConfig,
	}

}

func NewPositionCache(markets map[[32]byte]morpho.MarketParams) *PositionCache {
	bigMap := make(map[[32]byte]*Market, len(markets))

	for _, m := range markets {
		cache := make(map[common.Address]*BorrowPosition)
		bigMap[m.ID] = &Market{
			Mu: sync.RWMutex{},
			MarketCache: MarketCache{
				Oracle:    m.Oracle,
				Positions: cache,
			},
			MarketStats: MarketStats{
				LLTV: m.LLTV,
			},
		}
	}
	return &PositionCache{
		m: bigMap,
	}
}

// Overall loop logic

func (c *Cache) Init(conn *connector.Connector) error {
	err := c.ApiRefreshCache(conn.ClientHTTP)
	if err != nil {
		return err
	}
	arr := [][32]byte{}
	for k := range c.Config.Markets {
		arr = append(arr, k)
	}
	// REFRESH ON ALL MARKET FOR INIT
	c.OnChainRefresh(conn.ClientHTTP, arr)
	return nil
}

func (c *Cache) GetMarketProps(mId [32]byte) ([]*BorrowPosition, MarketStats) {
	market := c.PositionCache.m[mId]

	market.Mu.RLock()
	if market.TotalBorrowAssets == nil || market.TotalBorrowShares == nil {
		market.Mu.RUnlock()
		return nil, MarketStats{}
	}
	positions := make([]*BorrowPosition, 0, len(market.Positions))
	for _, p := range market.Positions {
		positions = append(positions, p)
	}
	stats := market.MarketStats
	market.Mu.RUnlock()
	return positions, stats

}

func (c *Cache) GetMarketPropsValue(mId [32]byte) ([]BorrowPosition, MarketStats) {
	market := c.PositionCache.m[mId]

	market.Mu.RLock()
	if market.TotalBorrowAssets == nil || market.TotalBorrowShares == nil {
		market.Mu.RUnlock()
		return nil, MarketStats{}
	}
	positions := make([]BorrowPosition, 0, len(market.Positions))
	for _, p := range market.Positions {
		positions = append(positions, *p)
	}
	stats := market.MarketStats
	market.Mu.RUnlock()
	return positions, stats

}

func (c *Cache) GetMorphoMarketFromId(mId [32]byte) morpho.MarketParams {
	return c.Config.Markets[mId]
}

// func (c *w3.Client) CallCtx(ctx context.Context, calls ...w3types.RPCCaller) error
func (c *Cache) EthCallCtx(client *w3.Client, ctx context.Context, calls []w3types.RPCCaller) error {
	c.LastMinCallCount.Add(int64(len(calls)))
	c.EthCallCount.Add(int64(len(calls)))
	return client.CallCtx(ctx, calls...)
}
