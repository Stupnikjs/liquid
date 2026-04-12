package onchain

import (
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/market"
	"github.com/Stupnikjs/morpho-sepolia/internal/state"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func ProcessEvents(c state.MarketReader, log *types.Log) {
	switch log.Topics[0] {
	case config.EventAccrueInterest.Topic0:
		AccrueInterestEventProcess(c, log)

	case config.EventBorrow.Topic0:
		BorrowEventProcess(c, log)

	case config.EventLiquidate.Topic0:
		LiquidateEventProcess(c, log)
	case config.EventRepay.Topic0:
		RepayEventProcess(c, log)
	case config.EventSupplyCollateral.Topic0:
		SupplyCollateralEventProcess(c, log)

	// ajouter les prix oracle
	default:
		fmt.Println("malformed log event")
	}
}

func BorrowEventProcess(c state.MarketReader, log *types.Log) {
	var (
		id       [32]byte
		caller   common.Address
		onBehalf common.Address
		receiver common.Address
		assets   big.Int
		shares   big.Int
	)
	if err := config.EventBorrow.DecodeArgs(log, &id, &caller, &onBehalf, &receiver, &assets, &shares); err != nil {
		fmt.Println("borrow ", log)
		fmt.Println("decode error:", err)
		return
	}

	if !slices.Contains(c.Ids(), id) {
		return
	}
	c.Update(id, func(m *market.Market) {
		if p, ok := m.Positions[onBehalf]; ok {
			if p.BorrowShares == nil {
				p.BorrowShares = new(big.Int)
				fmt.Println("borrowed:", p.Address)
			}
			p.BorrowShares.Add(p.BorrowShares, &shares)

		} else {
			m.Positions[onBehalf] = &market.BorrowPosition{
				MarketID:     id,
				Address:      onBehalf,
				BorrowShares: new(big.Int).Set(&shares),
			}
		}
	})

}

func RepayEventProcess(c state.MarketReader, log *types.Log) {
	var (
		id       [32]byte
		caller   common.Address
		onBehalf common.Address
		assets   big.Int
		shares   big.Int
	)

	if err := config.EventRepay.DecodeArgs(log, &id, &caller, &onBehalf, &assets, &shares); err != nil {
		fmt.Println("decode error:", err)
		return
	}
	if !slices.Contains(c.Ids(), id) {
		return
	}

	c.Update(id, func(m *market.Market) {
		if p, ok := m.Positions[onBehalf]; ok {
			p.BorrowShares.Sub(p.BorrowShares, &shares)
			if p.BorrowShares.Sign() <= 0 {
				delete(m.Positions, p.Address)
			}
		}
	})

}

func LiquidateEventProcess(c state.MarketReader, log *types.Log) {
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

	if err := config.EventLiquidate.DecodeArgs(log, &id, &caller, &borrower, &repaidAssets, &repaidShares, &seizedAssets, &badDebtAssets, &badDebtShares); err != nil {
		fmt.Println("liquidate ", log)
		fmt.Println("decode error:", err)
		return
	}

	if !slices.Contains(c.Ids(), id) {
		return
	}

	c.Update(id, func(m *market.Market) {
		if p, ok := m.Positions[borrower]; ok {
			p.BorrowShares.Sub(p.BorrowShares, &repaidShares)
			if p.BorrowShares.Sign() <= 0 {
				fmt.Println("borrow liquidated :", p.Address)
				delete(m.Positions, borrower)
			}
		}
	})

}

func AccrueInterestEventProcess(c state.MarketReader, log *types.Log) {
	var (
		id             [32]byte
		prevBorrowRate big.Int
		interest       big.Int
		feeShares      big.Int
	)
	if err := config.EventAccrueInterest.DecodeArgs(log, &id, &prevBorrowRate, &interest, &feeShares); err != nil {
		fmt.Println("decode error:", err)
		return
	}

	if !slices.Contains(c.Ids(), id) {
		return
	}

	// TotalBorrowAssets augmente des intérêts accumulés
	c.Update(id, func(m *market.Market) {

		if m.Stats.TotalBorrowAssets == nil {
			return
		}
		m.Stats.TotalBorrowAssets = new(big.Int).Add(m.Stats.TotalBorrowAssets, &interest)
		m.Stats.BorrowRate = &prevBorrowRate
		m.Stats.LastUpdate = time.Now().Unix()

	})

}

func SupplyCollateralEventProcess(c state.MarketReader, log *types.Log) {
	var (
		id       [32]byte
		caller   common.Address
		onBehalf common.Address
		assets   big.Int
	)
	if err := config.EventSupplyCollateral.DecodeArgs(log, &id, &caller, &onBehalf, &assets); err != nil {
		fmt.Println("decode error:", err)
		return
	}
	if !slices.Contains(c.Ids(), id) {
		return
	}

	c.Update(id, func(m *market.Market) {
		if p, ok := m.Positions[onBehalf]; ok {
			if p.CollateralAssets == nil {
				p.CollateralAssets = &assets
			} else {
				p.CollateralAssets.Add(p.CollateralAssets, &assets)
			}
		}
	})

}
