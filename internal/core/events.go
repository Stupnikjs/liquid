package core

import (
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/config"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (c *Cache) ProcessEvents(log *types.Log) {
	switch log.Topics[0] {
	case config.EventAccrueInterest.Topic0:
		c.PositionCache.AccrueInterestEventProcess(log)

	case config.EventBorrow.Topic0:
		c.PositionCache.BorrowEventProcess(log)

	case config.EventLiquidate.Topic0:
		c.PositionCache.LiquidateEventProcess(log)
	case config.EventRepay.Topic0:
		c.PositionCache.RepayEventProcess(log)
	case config.EventSupplyCollateral.Topic0:
		c.PositionCache.SupplyCollateralEventProcess(log)

	default:
		fmt.Println("malformed log event")
	}
}

func (c *PositionCache) BorrowEventProcess(log *types.Log) {
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
	if !c.IsMarketInCache(id) {
		return
	}
	market := c.m[id]
	market.Mu.Lock()
	if p, ok := market.Positions[onBehalf]; ok {
		if p.BorrowShares == nil {
			p.BorrowShares = new(big.Int)
			fmt.Println("borrowed:", p.Address)
		}
		p.BorrowShares.Add(p.BorrowShares, &shares)
	} else {
		market.Positions[onBehalf] = &BorrowPosition{
			MarketID:     id,
			Address:      onBehalf,
			BorrowShares: new(big.Int).Set(&shares),
		}
	}
	market.Mu.Unlock()
}

func (c *PositionCache) RepayEventProcess(log *types.Log) {
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
	if !c.IsMarketInCache(id) {
		return
	}
	market := c.m[id]
	market.Mu.Lock()
	if p, ok := market.Positions[onBehalf]; ok {
		p.BorrowShares.Sub(p.BorrowShares, &shares)
		if p.BorrowShares.Sign() <= 0 {
			delete(market.Positions, p.Address)
		}
	}
	market.Mu.Unlock()
}

func (c *PositionCache) LiquidateEventProcess(log *types.Log) {
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
	if !c.IsMarketInCache(id) {
		return
	}
	market := c.m[id]
	market.Mu.Lock()
	if p, ok := market.Positions[borrower]; ok {
		p.BorrowShares.Sub(p.BorrowShares, &repaidShares)
		if p.BorrowShares.Sign() <= 0 {
			fmt.Println("borrow liquidated :", p.Address)
			delete(market.Positions, borrower)
		}
	}
	market.Mu.Unlock()
}
func (c *PositionCache) AccrueInterestEventProcess(log *types.Log) {
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
	if !c.IsMarketInCache(id) {
		return
	}
	market := c.m[id]
	market.Mu.Lock()
	// TotalBorrowAssets augmente des intérêts accumulés

	if market.TotalBorrowAssets == nil {
		market.Mu.Unlock()
		return
	}
	market.TotalBorrowAssets = new(big.Int).Add(market.TotalBorrowAssets, &interest)
	market.BorrowRate = &prevBorrowRate
	market.LastUpdate = time.Now().Unix()
	market.Mu.Unlock()
}

func (c *PositionCache) SupplyCollateralEventProcess(log *types.Log) {
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
	if !c.IsMarketInCache(id) {
		return
	}
	market := c.m[id]
	market.Mu.Lock()
	if p, ok := market.Positions[onBehalf]; ok {
		if p.CollateralAssets == nil {
			p.CollateralAssets = &assets
		} else {
			p.CollateralAssets.Add(p.CollateralAssets, &assets)
		}
	}
	market.Mu.Unlock()
}

func (p *PositionCache) IsMarketInCache(marketID [32]byte) bool {
	market, ok := p.m[marketID]
	return ok && market != nil
}
