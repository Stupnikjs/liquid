package cache

import (
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/api"
	"github.com/ethereum/go-ethereum/common"
)

// Fecth positions from all markets in Ids()
// Fliter out too big pos and to small ones
// Sort by Collateral Value in USD

// Parsing to *BorrowPosition
func ApiItemToPos(p api.PositionItem, marketId [32]byte) *BorrowPosition {
	return &BorrowPosition{
		BorrowShares:     utils.ParseBigInt(p.State.BorrowShares.String()),
		BorrowAssetsUsd:  utils.ParseBigFloatToBigInt(p.State.BorrowAssetsUsd.String()),
		CollateralAssets: utils.ParseBigInt(p.State.Collateral.String()),
		MarketID:         marketId,
		Address:          common.HexToAddress(p.User.Address),
	}

}
