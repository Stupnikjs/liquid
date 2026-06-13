package onchain

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func ProcessEvents(c cache.MarketCache, log *types.Log, mabi *morpho.MorphoABI) {
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

func BorrowEventProcess(c cache.MarketCache, log *types.Log, mabi *morpho.MorphoABI) {
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

	c.UpdateBorrow(id, onBehalf, shares)
}

func RepayEventProcess(c cache.MarketCache, log *types.Log, mabi *morpho.MorphoABI) {
	id := [32]byte(log.Topics[1])
	if !slices.Contains(c.Ids(), id) {
		return
	}

	onBehalf := common.BytesToAddress(log.Topics[3].Bytes())

	out := map[string]interface{}{}
	if err := mabi.Events.Repay.Inputs.UnpackIntoMap(out, log.Data); err != nil {
		fmt.Println("repay decode error:", err)
		return
	}
	shares := out["shares"].(*big.Int)
	c.UpdateRepay(id, onBehalf, shares)

}

func LiquidateEventProcess(c cache.MarketCache, log *types.Log, mabi *morpho.MorphoABI) {
	id := [32]byte(log.Topics[1])
	if !slices.Contains(c.Ids(), id) {
		return
	}

	borrower := common.BytesToAddress(log.Topics[3].Bytes())

	out := map[string]any{}
	if err := mabi.Events.Liquidate.Inputs.UnpackIntoMap(out, log.Data); err != nil {
		fmt.Println("liquidate decode error:", err)
		return
	}
	repaidShares := out["repaidShares"].(*big.Int)
	badDebtShares := out["badDebtShares"].(*big.Int)
	c.UpdateLiquidate(id, borrower, repaidShares, badDebtShares)

}

func AccrueInterestEventProcess(c cache.MarketCache, log *types.Log, mabi *morpho.MorphoABI) {
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
	c.UpdateAccrueInterest(id, interest, prevBorrowRate)

}

func SupplyCollateralEventProcess(c cache.MarketCache, log *types.Log, mabi *morpho.MorphoABI) {
	id := [32]byte(log.Topics[1])
	if !slices.Contains(c.Ids(), id) {
		return
	}

	onBehalf := common.BytesToAddress(log.Topics[3].Bytes())

	out := map[string]any{}
	if err := mabi.Events.SupplyCollateral.Inputs.UnpackIntoMap(out, log.Data); err != nil {
		fmt.Println("supply collateral decode error:", err)
		return
	}
	assets := out["assets"].(*big.Int)
	c.UpdateSupplyCollateral(id, onBehalf, assets)

}

func WithdrawCollateralEventProcess(c *cache.MarketStore, log *types.Log, mabi *morpho.MorphoABI) {}
