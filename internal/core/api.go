package core

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"

	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/lmittmann/w3"
)

/* Api call */

func FetchBorrowersFromMarket(param morpho.MarketParams) ([]BorrowPosition, error) {
	ctx := context.Background()
	marketID := "0x" + hex.EncodeToString(param.ID[:])
	var result api.PositionsResult
	err := api.Query(ctx, api.PositionsQuery(marketID, param.ChainID), &result)
	if err != nil {
		return nil, fmt.Errorf("graphql fetch: %w", err)
	}

	return parsePositions(param, result), nil
}

func (c *Cache) ApiRefreshCache(client *w3.Client, threshold *big.Int) error {

	for _, ma := range c.Config.Markets {
		market := c.PositionCache.m[ma.ID]
		fetched, err := FetchBorrowersFromMarket(ma)
		if err != nil {
			return err
		}

		market.Mu.Lock()
		for _, p := range fetched {
			stats := market.MarketStats
			// filter here
			hf := p.HF(stats.TotalBorrowShares, stats.TotalBorrowAssets, stats.OraclePrice, ma.LLTV)
			// why not store hf in Liquidable struct
			if hf.Cmp(threshold) < 0 {
				market.Positions[p.Address] = &p
			}

		}
		market.Mu.Unlock()
	}

	return nil
}

func (c *Cache) LogHotMarket() {

	ctx := context.Background()

	var result api.MarketsResult
	err := api.Query(ctx, api.MarketsQuery(), &result)
	if err != nil {
		fmt.Printf("graphql fetch: %s", err.Error())
	}

	markets := result.Markets.Items
	sort.Slice(markets, func(i, j int) bool {
		return markets[i].CreationTimestamp > markets[j].CreationTimestamp
	})
	fmt.Println(markets[:100])
}

/*
type MarketCandidate struct {
    MarketID  [32]byte
    CreatedAt time.Time
    Params    morpho.MarketParams // collateral, loan, oracle, lltv
}

type MarketFilter struct {
    Candidates     map[[32]byte]*MarketCandidate
    MinBorrowUSD   float64 // ex: 500_000
    MinUniLiquidity float64 // ex: 200_000
    MaxAgeDays     int     // ex: 90
    Mu             sync.RWMutex
}

func (mf *MarketFilter) OnCreateMarket(event MarketCreatedEvent) {
    mf.Mu.Lock()
    defer mf.Mu.Unlock()
    mf.Candidates[event.MarketID] = &MarketCandidate{
        MarketID:  event.MarketID,
        CreatedAt: time.Now(),
        Params:    event.Params,
    }
}

func (mf *MarketFilter) ScanCandidates(client *w3.Client) []MarketCandidate {
    mf.Mu.RLock()
    defer mf.Mu.RUnlock()

    var approved []MarketCandidate

    for _, candidate := range mf.Candidates {
        // trop vieux
        if time.Since(candidate.CreatedAt).Hours() > float64(mf.MaxAgeDays*24) {
            delete(mf.Candidates, candidate.MarketID)
            continue
        }
        // check borrow TVL
        borrowAssets := fetchTotalBorrow(client, candidate.MarketID)
        if borrowAssets < mf.MinBorrowUSD {
            continue
        }
        // check liquidité Uniswap
        liquidity := fetchUniswapLiquidity(client, candidate.Params.CollateralToken, candidate.Params.LoanToken)
        if liquidity < mf.MinUniLiquidity {
            continue
        }
        approved = append(approved, *candidate)
    }
    return approved
}
*/
