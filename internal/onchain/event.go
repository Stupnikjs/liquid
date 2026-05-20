package onchain

import (
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"

	"github.com/Stupnikjs/liquid/internal/config/abi"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func ProcessEvents(c *cache.MarketStore, log *types.Log) {
	switch log.Topics[0] {
	case abi.EventAccrueInterest.Topic0:
		AccrueInterestEventProcess(c, log)

	case abi.EventBorrow.Topic0:
		BorrowEventProcess(c, log)

	case abi.EventLiquidate.Topic0:
		LiquidateEventProcess(c, log)
	case abi.EventRepay.Topic0:
		RepayEventProcess(c, log)
	case abi.EventSupplyCollateral.Topic0:
		SupplyCollateralEventProcess(c, log)

	// ajouter les prix oracle
	default:
		fmt.Println("malformed log event")
	}
}

func BorrowEventProcess(c *cache.MarketStore, log *types.Log) {
	var (
		id       [32]byte
		caller   common.Address
		onBehalf common.Address
		receiver common.Address
		assets   big.Int
		shares   big.Int
	)
	if err := abi.EventBorrow.DecodeArgs(log, &id, &caller, &onBehalf, &receiver, &assets, &shares); err != nil {
		fmt.Println("borrow ", log)
		fmt.Println("decode error:", err)
		return
	}

	if !slices.Contains(c.Ids(), id) {
		return
	}
	c.Update(id, func(m *cache.Market) {
		p := m.GetBorrowPosition(onBehalf)
		if p != nil {
			if p.BorrowShares == nil {
				p.BorrowShares = new(big.Int)
				fmt.Println("borrowed:", p.Address)
			}
			p.BorrowShares.Add(p.BorrowShares, &shares)

		} else {
			toInsert := &cache.BorrowPosition{
				MarketID:     id,
				Address:      onBehalf,
				BorrowShares: new(big.Int).Set(&shares),
			}
			m.InsertPositionUnsafe(toInsert)

		}
		// totalborrowshares cant be nil
		m.Stats.TotalBorrowShares.Add(m.Stats.TotalBorrowShares, &shares)
	})

}

func RepayEventProcess(c *cache.MarketStore, log *types.Log) {
	var (
		id       [32]byte
		caller   common.Address
		onBehalf common.Address
		assets   big.Int
		shares   big.Int
	)

	if err := abi.EventRepay.DecodeArgs(log, &id, &caller, &onBehalf, &assets, &shares); err != nil {
		fmt.Println("decode error:", err)
		return
	}
	if !slices.Contains(c.Ids(), id) {
		return
	}

	c.Update(id, func(m *cache.Market) {
		p := m.GetBorrowPosition(onBehalf)
		if p == nil {
			return
		}
		p.BorrowShares.Sub(p.BorrowShares, &shares)
		if p.BorrowShares.Sign() <= 0 {
			m.RemovePosition(onBehalf)
		}
		m.Stats.TotalBorrowShares.Sub(m.Stats.TotalBorrowShares, &shares)

	})

}

func LiquidateEventProcess(c *cache.MarketStore, log *types.Log) {
	var (
		id            [32]byte
		caller        common.Address
		borrower      common.Address
		repaidAssets  big.Int
		repaidShares  big.Int
		seizedAssets  big.Int
		badDebtAssets big.Int
		badDebtShares big.Int
	)

	if err := abi.EventLiquidate.DecodeArgs(log, &id, &caller, &borrower, &repaidAssets, &repaidShares, &seizedAssets, &badDebtAssets, &badDebtShares); err != nil {
		fmt.Println("liquidate ", log)
		fmt.Println("decode error:", err)
		return
	}

	if !slices.Contains(c.Ids(), id) {
		return
	}

	c.Update(id, func(m *cache.Market) {
		p := m.GetBorrowPosition(borrower)
		if p == nil {
			return
		}

		p.BorrowShares.Sub(p.BorrowShares, &repaidShares)
		if p.BorrowShares.Sign() <= 0 {
			fmt.Println("borrow liquidated :", p.Address)
			m.RemovePosition(borrower)
		}
		if m.Stats.TotalBorrowShares != nil {
			m.Stats.TotalBorrowShares.Sub(m.Stats.TotalBorrowShares, &repaidShares)
			// badDebtShares aussi réduisent le total
			m.Stats.TotalBorrowShares.Sub(m.Stats.TotalBorrowShares, &badDebtShares)
		}

	})

}

func AccrueInterestEventProcess(c *cache.MarketStore, log *types.Log) {
	var (
		id             [32]byte
		prevBorrowRate big.Int
		interest       big.Int
		feeShares      big.Int
	)
	if err := abi.EventAccrueInterest.DecodeArgs(log, &id, &prevBorrowRate, &interest, &feeShares); err != nil {
		fmt.Println("decode error:", err)
		return
	}

	if !slices.Contains(c.Ids(), id) {
		return
	}

	// TotalBorrowAssets augmente des intérêts accumulés
	c.Update(id, func(m *cache.Market) {

		if m.Stats.TotalBorrowAssets == nil {
			return
		}
		m.Stats.TotalBorrowAssets = new(big.Int).Add(m.Stats.TotalBorrowAssets, &interest)
		m.Stats.BorrowRate = &prevBorrowRate
		m.Stats.LastUpdate = time.Now().Unix()

	})

}

func SupplyCollateralEventProcess(c *cache.MarketStore, log *types.Log) {
	var (
		id       [32]byte
		caller   common.Address
		onBehalf common.Address
		assets   big.Int
	)
	if err := abi.EventSupplyCollateral.DecodeArgs(log, &id, &caller, &onBehalf, &assets); err != nil {
		fmt.Println("decode error:", err)
		return
	}
	if !slices.Contains(c.Ids(), id) {
		return
	}

	c.Update(id, func(m *cache.Market) {
		p := m.GetBorrowPosition(onBehalf)
		if p == nil {
			toInsert := &cache.BorrowPosition{
				MarketID:         id,
				Address:          onBehalf,
				CollateralAssets: new(big.Int).Set(&assets),
			}
			// to avoid deadlock
			m.InsertPositionUnsafe(toInsert)
		} else {
			if p.CollateralAssets == nil {
				p.CollateralAssets = &assets
			} else {
				p.CollateralAssets.Add(p.CollateralAssets, &assets)
			}
			// update position in sorted list
		}
	})

}

func WithdrawCollateralEventProcess(c *cache.MarketStore, log *types.Log) {}
