package cache

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/Stupnikjs/liquid/internal/utils"
)

func (s *MarketStore) GetSnapshot(id [32]byte) *MarketSnapshot {
	s.mu.RLock()
	market := s.markets[id]
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

func (s *MarketSnapshot) Analysis() MarketAnalysis {

	onedot1 := []BorrowPosition{}
	onedot5 := []BorrowPosition{}
	two := []BorrowPosition{}
	sup := []BorrowPosition{}
	totalUsdAtRisk := new(big.Int)
	for _, p := range s.Positions {
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

	fmt.Fprintf(&sb, "Positions: %d\n", a.FollowedPos)
	fmt.Fprintf(&sb, "   [1.0-1.1]: %d", a.Repartition[0])
	fmt.Fprintf(&sb, "   [1.1-1.5]: %d", a.Repartition[1])
	fmt.Fprintf(&sb, "   [1.5-2.0]: %d", a.Repartition[2])
	fmt.Fprintf(&sb, "   [>2.0]: %d\n", a.Repartition[3])
	fmt.Fprintf(&sb, "    USD less than 10 pct from liq: %s \n", utils.FormatWAD(a.TotalAtRiskUSD))
	return sb.String()
}
