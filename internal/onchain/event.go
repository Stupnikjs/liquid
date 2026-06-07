package onchain

import (
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func ProcessEvents(c *cache.MarketStore, log *types.Log, mabi *morpho.MorphoABI) {
	if len(log.Topics) == 0 {
		return
	}
	switch log.Topics[0] {
	case mabi.Topics.AccrueInterest:
		AccrueInterestEventProcess(c, log, mabi)
	case mabi.Topics.Borrow:
		BorrowEventProcess(c, log, mabi)
	case mabi.Topics.Liquidate:
		LiquidateEventProcess(c, log, mabi)
	case mabi.Topics.Repay:
		RepayEventProcess(c, log, mabi)
	case mabi.Topics.SupplyCollateral:
		SupplyCollateralEventProcess(c, log, mabi)
	default:
		fmt.Println("malformed log event")
	}
}

func BorrowEventProcess(c *cache.MarketStore, log *types.Log, mabi *morpho.MorphoABI) {
	// indexed → topics
	id := [32]byte(log.Topics[1])
	if !slices.Contains(c.Ids(), id) {
		return
	}
	onBehalf := common.BytesToAddress(log.Topics[2].Bytes())

	// non-indexed → Data
	out := map[string]interface{}{}
	if err := mabi.Events.Borrow.Inputs.UnpackIntoMap(out, log.Data); err != nil {
		fmt.Println("borrow decode error:", err)
		return
	}
	shares := out["shares"].(*big.Int)

	c.Update(id, func(m *cache.Market) {
		p := m.GetBorrowPosition(onBehalf)
		if p != nil {
			if p.BorrowShares == nil {
				p.BorrowShares = new(big.Int)
			}
			p.BorrowShares.Add(p.BorrowShares, shares)
		} else {
			m.InsertPositionUnsafe(&cache.BorrowPosition{
				MarketID:     id,
				Address:      onBehalf,
				BorrowShares: new(big.Int).Set(shares),
			})
		}
		m.Stats.TotalBorrowShares.Add(m.Stats.TotalBorrowShares, shares)
	})
}

func RepayEventProcess(c *cache.MarketStore, log *types.Log, mabi *morpho.MorphoABI) {
	id := [32]byte(log.Topics[1])
	if !slices.Contains(c.Ids(), id) {
		return
	}

	onBehalf := common.BytesToAddress(log.Topics[2].Bytes())

	out := map[string]interface{}{}
	if err := mabi.Events.Repay.Inputs.UnpackIntoMap(out, log.Data); err != nil {
		fmt.Println("repay decode error:", err)
		return
	}
	shares := out["shares"].(*big.Int)

	c.Update(id, func(m *cache.Market) {
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

func LiquidateEventProcess(c *cache.MarketStore, log *types.Log, mabi *morpho.MorphoABI) {
	id := [32]byte(log.Topics[1])
	if !slices.Contains(c.Ids(), id) {
		return
	}

	borrower := common.BytesToAddress(log.Topics[2].Bytes())

	out := map[string]interface{}{}
	if err := mabi.Events.Liquidate.Inputs.UnpackIntoMap(out, log.Data); err != nil {
		fmt.Println("liquidate decode error:", err)
		return
	}
	repaidShares := out["repaidShares"].(*big.Int)
	badDebtShares := out["badDebtShares"].(*big.Int)

	c.Update(id, func(m *cache.Market) {
		p := m.GetBorrowPosition(borrower)
		if p == nil {
			return
		}
		p.BorrowShares.Sub(p.BorrowShares, repaidShares)
		if p.BorrowShares.Sign() <= 0 {
			fmt.Println("borrow liquidated :", p.Address)
			m.RemovePosition(borrower)
		}
		if m.Stats.TotalBorrowShares != nil {
			m.Stats.TotalBorrowShares.Sub(m.Stats.TotalBorrowShares, repaidShares)
			m.Stats.TotalBorrowShares.Sub(m.Stats.TotalBorrowShares, badDebtShares)
		}
	})
}

func AccrueInterestEventProcess(c *cache.MarketStore, log *types.Log, mabi *morpho.MorphoABI) {
	id := [32]byte(log.Topics[1])
	if !slices.Contains(c.Ids(), id) {
		return
	}

	out := map[string]interface{}{}
	if err := mabi.Events.AccrueInterest.Inputs.UnpackIntoMap(out, log.Data); err != nil {
		fmt.Println("accrue interest decode error:", err)
		return
	}
	interest := out["interest"].(*big.Int)
	prevBorrowRate := out["prevBorrowRate"].(*big.Int)

	c.Update(id, func(m *cache.Market) {
		if m.Stats.TotalBorrowAssets == nil {
			return
		}
		m.Stats.TotalBorrowAssets.Add(m.Stats.TotalBorrowAssets, interest)
		m.Stats.BorrowRate = prevBorrowRate
		m.Stats.LastUpdate = time.Now().Unix()
	})
}

func SupplyCollateralEventProcess(c *cache.MarketStore, log *types.Log, mabi *morpho.MorphoABI) {
	id := [32]byte(log.Topics[1])
	if !slices.Contains(c.Ids(), id) {
		return
	}

	onBehalf := common.BytesToAddress(log.Topics[2].Bytes())

	out := map[string]interface{}{}
	if err := mabi.Events.SupplyCollateral.Inputs.UnpackIntoMap(out, log.Data); err != nil {
		fmt.Println("supply collateral decode error:", err)
		return
	}
	assets := out["assets"].(*big.Int)

	c.Update(id, func(m *cache.Market) {
		p := m.GetBorrowPosition(onBehalf)
		if p == nil {
			m.InsertPositionUnsafe(&cache.BorrowPosition{
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

func WithdrawCollateralEventProcess(c *cache.MarketStore, log *types.Log, mabi *morpho.MorphoABI) {}
