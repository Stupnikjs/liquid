package morpho

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var IRM = common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687")

type ChainConfig struct {
	WalletAddress        common.Address
	LiquidatorAddress    common.Address
	UniswapRouterAddress common.Address
	Signer               *Signer
	Name                 string
}

type MarketParams struct {
	ID              [32]byte       // 32
	LoanToken       common.Address // 20
	CollateralToken common.Address // 20
	Oracle          common.Address // 20
	// 4 padding
	LLTV                    *big.Int // 8
	LoanTokenStr            string   // 16
	CollateralTokenStr      string   // 16
	ChainID                 uint32   // 4
	PoolFee                 int32    // 4 ← change int→int32, suffisant pour un fee
	LoanTokenDecimals       uint16   // 2
	CollateralTokenDecimals uint16   // 2
	Correlated              bool     // 1
	CexOnly                 bool
	// 3 padding
}
type MarketContractParams struct {
	LoanToken       common.Address
	CollateralToken common.Address
	Oracle          common.Address
	Irm             common.Address
	Lltv            *big.Int
}

func (m *MarketParams) ToMarketContractParams() *MarketContractParams {
	return &MarketContractParams{
		LoanToken:       m.LoanToken,
		CollateralToken: m.CollateralToken,
		Oracle:          m.Oracle,
		Irm:             IRM,
		Lltv:            m.LLTV,
	}
}

func NotCexOnlyMarket(params []MarketParams) []MarketParams {
	arr := []MarketParams{}
	for _, m := range params {
		if m.CexOnly {
			continue
		}
		arr = append(arr, m)
	}
	return arr
}

func NotCexOnlyIds(params []MarketParams) [][32]byte {
	var arr [][32]byte
	for _, m := range params {
		if m.CexOnly {
			continue
		}
		arr = append(arr, m.ID)
	}
	return arr
}
