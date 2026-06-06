package morpho

import (
	"math/big"

	"github.com/Stupnikjs/liquid/internal/utils"
)

var WAD = utils.TenPowInt(18)

// liquidationIncentiveFactor reproduit la formule Morpho :
// LIF = min(maxLIF, WAD / LLTV - WAD)  avec maxLIF = 0.15e18
func LiquidationIncentiveFactor(lltv *big.Int) *big.Int {
	maxLIF := new(big.Int).Div(WAD, big.NewInt(100))
	maxLIF.Mul(maxLIF, big.NewInt(15)) // 0.15 WAD

	// WAD / LLTV - WAD
	lif := new(big.Int).Mul(WAD, WAD)
	lif.Div(lif, lltv)
	lif.Sub(lif, WAD)

	if lif.Cmp(maxLIF) > 0 {
		return maxLIF
	}
	return lif
}

func BorrowAssetsFromShares(posShares, totShares, totBorrowAssets *big.Int) *big.Int {
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
		new(big.Int).Mul(posShares, totBorrowAssets),
		totShares)
}

// sharesToAssets : shares * totalAssets / totalShares
func SharesToAssets(shares, totalAssets, totalShares *big.Int) *big.Int {
	if totalShares.Sign() == 0 {
		return new(big.Int)
	}
	r := new(big.Int).Mul(shares, totalAssets)
	return r.Div(r, totalShares)
}

/* Profit = seized - repayed - gas */
func EstimateProfit(
	seizeAssets, repayAssets *big.Int,
	gasEst uint64,
) *big.Int {
	// gasCost en wei (gasEst * gasPrice)
	// TODO : récupérer gasPrice dynamiquement via eth_gasPrice
	gasPrice := big.NewInt(3e9) // 3 gwei placeholder
	gasCostWei := new(big.Int).Mul(big.NewInt(int64(gasEst)), gasPrice)

	// Profit brut = seizeAssets - repayAssets (en collateral token)
	// Pour comparer avec gasCostWei il faut convertir en ETH via oracle
	// Simplifié ici : on suppose collateral = ETH-like, à affiner
	profit := new(big.Int).Sub(seizeAssets, repayAssets)
	profit.Sub(profit, gasCostWei)

	return profit
}

func ComputeSeizedAsset(BorrowShares, TotalBorrowAssets, TotalBorrowShares, LLTV *big.Int) (*big.Int, *big.Int) {
	
	// repayAssets = repayShares * totalBorrowAssets / totalBorrowShares
	repayAssets := SharesToAssets(repayShares, TotalBorrowAssets, TotalBorrowShares)

	// seizeAssets = repayAssets * (WAD + liquidationIncentiveFactor) / WAD
	// liquidationIncentiveFactor dépend de LLTV (voir Morpho docs)
	lif := LiquidationIncentiveFactor(LLTV)
	seizeAssets := new(big.Int).Mul(repayAssets, new(big.Int).Add(WAD, lif))
	seizeAssets.Div(seizeAssets, WAD)

	return seizeAssets
}

func ComputeMinOut(seizedAssets, collateralPrice *big.Int) *big.Int {
	valueInLoan := new(big.Int).Mul(seizedAssets, collateralPrice)
	return valueInLoan.Div(valueInLoan, utils.TenPowInt(36))
}

// si pblm reprendre fonction historique dans cache
func HF(CollateralAssets, BorrowShares, TotalBorrowShares, TotalBorrowAssets, LLTV, OraclePrice *big.Int) *big.Int {

	borrowAssets := BorrowAssetsFromShares(
		BorrowShares, TotalBorrowShares, TotalBorrowAssets,
	)
	if borrowAssets == nil || CollateralAssets == nil {
		return nil // ← nil, not 0
	}
	if borrowAssets.Sign() == 0 || CollateralAssets.Sign() == 0 {
		return nil // ← nil, not 0
	}
	// numerator = collateral * price * LLTV
	// price of collateral in loan token 
	numerator := new(big.Int).Mul(CollateralAssets, OraclePrice)
	numerator.Mul(numerator, LLTV)
	// denominator = borrow * 1e36
	denominator := new(big.Int).Mul(borrowAssets, utils.TenPowInt(36))
	hf := new(big.Int).Div(numerator, denominator)
	// utils.BigIntToFloat(hf)/1e18
	return hf

}
