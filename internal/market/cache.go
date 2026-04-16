package market

import (
	"fmt"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/Stupnikjs/morpho-sepolia/pkg/swap"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

type Cache struct {
	Markets   *MarketStore
	MarketMap map[[32]byte]morpho.MarketParams
}

func NewCache(conn *connector.Connector, conf config.Config, filters api.MarketFilters) *Cache {
	result, err := api.QueryMarkets(conn.ClientHTTP, conf.ChainID)
	if err != nil {
		return nil
	}

	markets := api.FilterMarket(result, filters, conf.ChainID)

	for _, m := range markets {
		swapRes, _ := swap.BestQuote(conn, m.CollateralToken, m.LoanToken, utils.TenPowInt(5))
		res, _ := swapRes.AmountOut.Float64()
		slip := 1 - (res / 10_000)
		fmt.Println(m.CollateralTokenStr, m.LoanTokenStr, slip)
	}

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

func ApiCall(client *w3.Client, marketStore *MarketStore, marketMap map[[32]byte]morpho.MarketParams, chainId uint32) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for id := range marketMap {
		wg.Add(1)
		go func(id [32]byte) {
			defer wg.Done()

			fetched, err := api.FetchBorrowersFromMarket(id, chainId)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			for _, p := range ApiItemToPos(fetched, id) {

				marketStore.Update(id, func(m *Market) {
					m.Positions[p.Address] = p
				}) // capture loop variable

			}
		}(id)
	}

	wg.Wait()
	return firstErr
}

func ApiItemToPos(result api.PositionsResult, marketId [32]byte) []*BorrowPosition {
	var positions []*BorrowPosition
	for _, p := range result.MarketPositions.Items {
		pos := &BorrowPosition{
			BorrowShares:     utils.ParseBigInt(p.State.BorrowShares.String()),
			CollateralAssets: utils.ParseBigInt(p.State.BorrowShares.String()),
			MarketID:         marketId,
			Address:          common.HexToAddress(p.User.Address),
		}
		positions = append(positions, pos)
	}
	return positions
}
