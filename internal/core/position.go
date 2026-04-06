package core

import (
	"math/big"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"

	"github.com/ethereum/go-ethereum/common"
)

type BorrowPosition struct {
	MarketID                       [32]byte
	Address                        common.Address
	BorrowShares, CollateralAssets *big.Int
	SimulationCount                int
	Attempts                       int
	LastAttempt                    int64
}

// AccruedBorrowAssets retourne totalBorrowAssets mis à jour jusqu'à `now`
// To Simulate Morpho call accrue interest at begin of liquidate func
func (ms *MarketStats) AccruedBorrowAssets() *big.Int {
	if ms.TotalBorrowAssets == nil {
		return ms.TotalBorrowAssets
	}
	if ms.TotalBorrowAssets.Sign() == 0 {
		return ms.TotalBorrowAssets
	}

	dt := big.NewInt(time.Now().Unix() - ms.LastUpdate)
	if dt.Sign() <= 0 {
		return ms.TotalBorrowAssets
	}
	if ms.BorrowRate == nil {
		return ms.TotalBorrowAssets
	}
	// interest = totalBorrowAssets * borrowRate * dt / WAD
	interest := new(big.Int).Mul(ms.TotalBorrowAssets, ms.BorrowRate)
	interest.Mul(interest, dt)
	interest.Div(interest, utils.WAD)

	return new(big.Int).Add(ms.TotalBorrowAssets, interest)
}

func (pos *BorrowPosition) GetBorrowAssets(totShares, totBorrowAssets *big.Int) *big.Int {
	if totBorrowAssets == nil || totShares == nil {
		return new(big.Int)
	}
	if totShares.Sign() == 0 {
		return new(big.Int)
	}
	if totBorrowAssets.Sign() == 0 {
		return new(big.Int)
	}
	return new(big.Int).Div(
		new(big.Int).Mul(pos.BorrowShares, totBorrowAssets),
		totShares)
}

// prec 1e18
func (pos *BorrowPosition) HF(totShares, totBorrowAssets, oraclePrice, LLTV *big.Int) *big.Int {
	borrowAssets := pos.GetBorrowAssets(totShares, totBorrowAssets)
	if borrowAssets == nil || pos.CollateralAssets == nil {
		return big.NewInt(0)
	}
	if borrowAssets.Sign() == 0 || pos.CollateralAssets.Sign() == 0 {
		return big.NewInt(0)
	}

	hf := new(big.Int).Div(
		new(big.Int).Mul(pos.CollateralAssets, oraclePrice),
		borrowAssets)

	return new(big.Int).Div(
		new(big.Int).Mul(hf, LLTV),
		utils.TenPowInt(36),
	)
}

func parsePositions(params morpho.MarketParams, result api.GraphQLResult) []BorrowPosition {
	items := result.Data.MarketPositions.Items
	positions := make([]BorrowPosition, 0, len(items))

	for _, item := range items {
		borrowShares := utils.ParseBigInt(item.State.BorrowShares.String())
		collateral := utils.ParseBigInt(item.State.Collateral.String())

		// ignore les positions fermées
		if borrowShares.Sign() == 0 && collateral.Sign() == 0 {
			continue
		}

		positions = append(positions, BorrowPosition{
			MarketID:         params.ID,
			Address:          common.HexToAddress(item.User.Address),
			BorrowShares:     borrowShares,
			CollateralAssets: collateral,
		})
	}
	return positions
}

// need to pre compute Tx and modify nonce based on the block to optimize speed

func (pos *BorrowPosition) EstimateProfit(
	totShares, totBorrowAssets, oraclePrice *big.Int,
	m morpho.MarketParams,
) *big.Int {
	borrowAssets := pos.GetBorrowAssets(totShares, totBorrowAssets)
	if borrowAssets.Sign() == 0 || pos.CollateralAssets.Sign() == 0 {
		return big.NewInt(0)
	}

	// collatéral saisi converti en loan token via oracle
	// oracle price = collateral/loan * 1e36
	// collateralValueInLoan = collateralAssets * oraclePrice / 1e36
	collateralValueInLoan := new(big.Int).Div(
		new(big.Int).Mul(pos.CollateralAssets, oraclePrice),
		utils.TenPowInt(36),
	)

	// ajuste les décimales
	// collateral → loan token
	isCollateralMorePrecise := m.CollateralTokenDecimals > m.LoanTokenDecimals
	if isCollateralMorePrecise {
		diff := m.CollateralTokenDecimals - m.LoanTokenDecimals
		collateralValueInLoan.Div(collateralValueInLoan, utils.TenPowInt(uint(diff)))
	} else {
		diff := m.LoanTokenDecimals - m.CollateralTokenDecimals
		collateralValueInLoan.Mul(collateralValueInLoan, utils.TenPowInt(uint(diff)))
	}

	profit := new(big.Int).Sub(collateralValueInLoan, borrowAssets)
	if profit.Sign() < 0 {
		return big.NewInt(0)
	}

	return profit
}
