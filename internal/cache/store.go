package cache

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/ethereum/go-ethereum/common"
)

func (s *Cache) Ids() [][32]byte {
	s.mu.RLock()
	ids := make([][32]byte, 0, len(s.Markets.markets))
	for id, m := range s.Markets.markets {
		if !m.Canceled {
			ids = append(ids, id)
		}
	}
	s.mu.RUnlock()
	return ids

}

// need to unexport update for interface
func (s *MarketStore) Update(id [32]byte, fn func(m *Market)) {
	s.mu.RLock()
	m := s.markets[id]
	s.mu.RUnlock()

	if m == nil {
		return
	}

	m.Mu.Lock()
	fn(m)
	m.Mu.Unlock()
}

// errase positions by new positions array
func (c *Cache) UpdatePositionsSlice(id [32]byte, positions []*BorrowPosition) {
	c.Markets.Update(id, func(m *Market) {
		m.Positions = positions
	})
}

func (c *Cache) UpdateMarketStats(id [32]byte, stats MarketStats) {
	c.Markets.Update(id, func(m *Market) {
		m.Stats = stats
	})
}

func (c *Cache) UpdateOnchainRefresh(id [32]byte, totalBorrowShares, totalBorrowAssets, oraclePrice *big.Int) {
	c.Markets.Update(id, func(m *Market) {
		m.Stats.TotalBorrowAssets = totalBorrowAssets
		m.Stats.TotalBorrowShares = totalBorrowShares
		m.Oracle.Price = oraclePrice
	})
}

func (c *Cache) UpdateOraclePrice(id [32]byte, oraclePrice *big.Int) {
	c.Markets.Update(id, func(m *Market) {
		m.Oracle.Price = oraclePrice
	})
}

func (c *Cache) UpdateMaxCollateralPos(id [32]byte, MaxCollateralPos *big.Int) {
	c.Markets.Update(id, func(m *Market) {
		m.Stats.MaxCollateralPos = MaxCollateralPos
	})
}

func (c *Cache) UpdateAccrueInterest(id [32]byte, interest, prevBorrowRate *big.Int) {
	c.Markets.Update(id, func(m *Market) {
		if m.Stats.TotalBorrowAssets == nil {
			return
		}
		m.Stats.TotalBorrowAssets.Add(m.Stats.TotalBorrowAssets, interest)
		m.Stats.BorrowRate = prevBorrowRate
	})
}

func (c *Cache) UpdateRepay(id [32]byte, onBehalf common.Address, shares *big.Int) {
	c.Markets.Update(id, func(m *Market) {
		p := m.GetBorrowPosition(onBehalf)
		if p == nil {
			return
		}
		p.BorrowShares.Sub(p.BorrowShares, shares)
		if p.BorrowShares.Sign() <= 0 {
			m.RemovePosition(onBehalf)
		}
		m.Stats.TotalBorrowShares.Sub(m.Stats.TotalBorrowShares, shares)
	})
}

func (c *Cache) UpdateBorrow(id [32]byte, onBehalf common.Address, shares *big.Int) {
	c.Markets.Update(id, func(m *Market) {
		p := m.GetBorrowPosition(onBehalf)
		if p != nil {
			if p.BorrowShares == nil {
				p.BorrowShares = new(big.Int)
			}
			p.BorrowShares.Add(p.BorrowShares, shares)
		} else {
			m.InsertPositionUnsafe(&BorrowPosition{
				MarketID:     id,
				Address:      onBehalf,
				BorrowShares: new(big.Int).Set(shares),
			})
		}
		m.Stats.TotalBorrowShares.Add(m.Stats.TotalBorrowShares, shares)
	})
}

func (c *Cache) UpdateSupplyCollateral(id [32]byte, onBehalf common.Address, assets *big.Int) {
	c.Markets.Update(id, func(m *Market) {
		p := m.GetBorrowPosition(onBehalf)
		if p == nil {
			m.InsertPositionUnsafe(&BorrowPosition{
				MarketID:         id,
				Address:          onBehalf,
				CollateralAssets: new(big.Int).Set(assets),
			})
		} else {
			if p.CollateralAssets == nil {
				p.CollateralAssets = new(big.Int).Set(assets)
			} else {
				p.CollateralAssets.Add(p.CollateralAssets, assets)
			}
		}
	})
}

func (c *Cache) UpdateLiquidate(id [32]byte, onBehalf common.Address, repaidShares, badDebtShares *big.Int) {
	c.Markets.Update(id, func(m *Market) {
		p := m.GetBorrowPosition(onBehalf)
		if p == nil {
			return
		}
		p.BorrowShares.Sub(p.BorrowShares, repaidShares)
		if p.BorrowShares.Sign() <= 0 {
			fmt.Println("borrow liquidated :", p.Address)
			m.RemovePosition(onBehalf)
		}
		if m.Stats.TotalBorrowShares != nil {
			m.Stats.TotalBorrowShares.Sub(m.Stats.TotalBorrowShares, repaidShares)
			m.Stats.TotalBorrowShares.Sub(m.Stats.TotalBorrowShares, badDebtShares)
		}
	})
}

func (c *Cache) UpdateRecompute(id [32]byte, tickCount int) {
	c.Markets.Update(id, func(m *Market) {
		m.RecomputeHFUnsafe(len(m.Positions) / 2)
		if tickCount%10 == 0 {
			m.RecomputeHFUnsafe(len(m.Positions))
			m.SortAllPositionsByHFUnsafe()
		}
	})
}

func (c *Cache) CancelMarket(id [32]byte) {
	c.Markets.Update(id, func(m *Market) {
		m.Canceled = true
	})
}

func (s *Cache) GetSnapshot(id [32]byte) *MarketSnapshot {
	s.mu.RLock()
	market := s.Markets.markets[id]
	s.mu.RUnlock()
	if market == nil {
		return nil
	}
	market.Mu.RLock()
	defer market.Mu.RUnlock()
	if market.Canceled ||
		market.Oracle.Price == nil ||
		market.LLTV == nil ||
		market.Stats.TotalBorrowAssets == nil ||
		market.Stats.TotalBorrowShares == nil || market.Stats.MaxCollateralPos == nil {
		return nil
	}

	snap := &MarketSnapshot{
		ID: id,
		Oracle: Oracle{
			Price:   new(big.Int).Set(market.Oracle.Price),
			Address: market.Oracle.Address,
		},
		LLTV: new(big.Int).Set(market.LLTV),
		Stats: MarketStats{
			TotalBorrowAssets: new(big.Int).Set(market.Stats.TotalBorrowAssets),
			TotalBorrowShares: new(big.Int).Set(market.Stats.TotalBorrowShares),
			MaxCollateralPos:  new(big.Int).Set(market.Stats.MaxCollateralPos),
		},
		Positions: make([]BorrowPosition, 0, len(market.Positions)),
	}
	// anyway to avoid loop
	for _, p := range market.Positions {
		snap.Positions = append(snap.Positions, *p)
	}

	return snap
}

type MarketAnalysis struct {
	FollowedPos    int
	Repartition    [4]int
	Utilization    float64  // TotalBorrow / TotalSupply
	TotalAtRiskUSD *big.Int // valeur totale HF < 1.1
}

func (c *Cache) LogMarkets() {
	ids := c.Ids()
	for _, id := range ids {
		m := c.MarketMap[id]
		snap := c.GetSnapshot(id)
		if snap == nil {
			continue
		}
		fmt.Printf("[%s/%s]", m.CollateralTokenStr, m.LoanTokenStr)
	}
}

func (s *MarketSnapshot) Analysis() MarketAnalysis {

	onedot1 := []BorrowPosition{}
	onedot5 := []BorrowPosition{}
	two := []BorrowPosition{}
	sup := []BorrowPosition{}
	totalUsdAtRisk := new(big.Int)
	for _, p := range s.Positions {
		if p.CachedHF == nil || p.CachedHF.Sign() == 0 {
			continue
		}
		if p.CachedHF.Cmp(utils.WAD1DOT1) < 0 {
			onedot1 = append(onedot1, p)
			if p.BorrowAssetsUsd == nil {
				continue
			}
			totalUsdAtRisk.Add(totalUsdAtRisk, p.BorrowAssetsUsd)
		} else if p.CachedHF.Cmp(utils.WAD1DOT5) < 0 {
			onedot5 = append(onedot5, p)
		} else if p.CachedHF.Cmp(utils.WAD_2) < 0 {
			two = append(two, p)
		} else {
			sup = append(sup, p)
		}

	}
	repart := [4]int{len(onedot1), len(onedot5), len(two), len(sup)}
	return MarketAnalysis{
		FollowedPos:    len(s.Positions),
		Repartition:    repart,
		TotalAtRiskUSD: totalUsdAtRisk,
	}

}

func (a MarketAnalysis) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Pos: %d", a.FollowedPos)
	fmt.Fprintf(&sb, " [1.0-1.1]: %d", a.Repartition[0])
	fmt.Fprintf(&sb, " [1.1-1.5]: %d", a.Repartition[1])
	// fmt.Fprintf(&sb, " [1.5-2.0]: %d", a.Repartition[2])
	//fmt.Fprintf(&sb, " [>2.0]: %d \n", a.Repartition[3])
	fmt.Fprintf(&sb, " USD < 10pct from liq: %s \n", utils.FormatWAD(a.TotalAtRiskUSD))
	return sb.String()
}
