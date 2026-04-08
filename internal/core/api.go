package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/lmittmann/w3"
)

/* Api call */

func FetchBorrowersFromMarket(param morpho.MarketParams) ([]BorrowPosition, error) {
	marketID := "0x" + hex.EncodeToString(param.ID[:])

	data, err := api.GraphqlPost(api.PositionsQuery(marketID, param.ChainID))
	if err != nil {
		return nil, fmt.Errorf("graphql fetch: %w", err)
	}

	var result api.GraphQLResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("graphql decode: %w", err)
	}

	return parsePositions(param, result), nil
}

func (c *Cache) ApiRefreshCache(client *w3.Client) error {

	for _, ma := range c.Config.Markets {
		market := c.PositionCache.m[ma.ID]
		fetched, err := FetchBorrowersFromMarket(ma)
		if err != nil {
			return err
		}

		market.Mu.Lock()
		for _, p := range fetched {
   // filtre ici 
			market.Positions[p.Address] = &p

		}
		market.Mu.Unlock()
	}

	return nil
}


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
