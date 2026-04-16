package market

import (
	"math/big"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
	"github.com/Stupnikjs/morpho-sepolia/pkg/api"
	"github.com/Stupnikjs/morpho-sepolia/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
)

type BorrowPosition struct {
	MarketID                       [32]byte
	Address                        common.Address
	BorrowShares, CollateralAssets *big.Int
}

// prec 1e18
func (pos *BorrowPosition) HF(
	totShares, totBorrowAssets, oraclePrice, LLTV *big.Int,
) *big.Int {
	borrowAssets := morpho.BorrowAssetsFromShares(
		pos.BorrowShares, totShares, totBorrowAssets,
	)
	if borrowAssets == nil || pos.CollateralAssets == nil {
		return big.NewInt(0)
	}
	if borrowAssets.Sign() == 0 || pos.CollateralAssets.Sign() == 0 {
		return big.NewInt(0)
	}
	// numerator = collateral * price * LLTV
	numerator := new(big.Int).Mul(pos.CollateralAssets, oraclePrice)
	numerator.Mul(numerator, LLTV)
	// denominator = borrow * 1e36
	denominator := new(big.Int).Mul(borrowAssets, utils.TenPowInt(36))
	hf := new(big.Int).Div(numerator, denominator)
	// utils.BigIntToFloat(hf)/1e18
	return hf
}

// need to pre compute Tx and modify nonce based on the block to optimize speed

func (pos *BorrowPosition) EstimateProfit(
	totShares, totBorrowAssets, oraclePrice *big.Int,
	m morpho.MarketParams,
) *big.Int {
	borrowAssets := morpho.BorrowAssetsFromShares(pos.BorrowShares, totShares, totBorrowAssets)
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

func ParsePositions(id [32]byte, result api.PositionsResult) []BorrowPosition {
	items := result.MarketPositions.Items // ✅ plus de .Data
	positions := make([]BorrowPosition, 0, len(items))
	for _, item := range items {
		borrowAssetUsd := utils.ParseBigInt(item.State.BorrowAssetsUsd.String())
		if borrowAssetUsd.Cmp(utils.TenPowInt(2)) < 0 {
			continue
		}
		borrowShares := utils.ParseBigInt(item.State.BorrowShares.String())
		collateral := utils.ParseBigInt(item.State.Collateral.String())
		if borrowShares.Sign() == 0 && collateral.Sign() == 0 {
			continue
		}
		positions = append(positions, BorrowPosition{
			MarketID:         id,
			Address:          common.HexToAddress(item.User.Address),
			BorrowShares:     borrowShares,
			CollateralAssets: collateral,
		})
	}
	return positions
}
