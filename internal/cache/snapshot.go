package cache

import (
	"fmt"

	"github.com/Stupnikjs/liquid/internal/utils"
)

type MarketAnalysis struct {
	FollowedPos int
	Repartition [4]int // num of pos with hf in 1 - 1.1 range 1.1-1.5 range 1.5-2.0 and > 2 asuming noliquidable pos

}

func (s *MarketSnapshot) Analysis() MarketAnalysis {

	onedot1 := 0
	onedot5 := 0
	two := 0
	sup := 0
	for _, p := range s.Positions {
		if p.CachedHF.Cmp(utils.WAD1DOT1) < 0 {
			onedot1++
		} else if p.CachedHF.Cmp(utils.WAD1DOT5) < 0 {
			onedot5++
		} else if p.CachedHF.Cmp(utils.WAD_2) < 0 {
			two++
		} else {
			sup++
		}

	}
	repart := [4]int{onedot1, onedot5, two, sup}
	return MarketAnalysis{
		FollowedPos: len(s.Positions),
		Repartition: repart,
	}

}

func (a MarketAnalysis) String() string {
	return fmt.Sprintf(
		"Positions: %d | [1.0-1.1]: %d | [1.1-1.5]: %d | [1.5-2.0]: %d | [>2.0]: %d",
		a.FollowedPos,
		a.Repartition[0],
		a.Repartition[1],
		a.Repartition[2],
		a.Repartition[3],
	)
}
