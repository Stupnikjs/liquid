package morpho

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

var IRM = common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687")

type MarketParams struct {
	ID              [32]byte       // 32
	LoanToken       common.Address // 20
	CollateralToken common.Address // 20
	Oracle          common.Address // 20
	// 4 padding
	Irm                     common.Address
	LLTV                    *big.Int // 8
	LoanTokenStr            string   // 16
	CollateralTokenStr      string   // 16
	ChainID                 uint32   // 4
	PoolFee                 int32    // 4 ← change int→int32, suffisant pour un fee
	LoanTokenDecimals       uint16   // 2
	CollateralTokenDecimals uint16   // 2
	// 3 padding
}

type MarketContractParams struct {
	LoanToken       common.Address `abi:"loanToken"`
	CollateralToken common.Address `abi:"collateralToken"`
	Oracle          common.Address `abi:"oracle"`
	Irm             common.Address `abi:"irm"`
	LLTV            *big.Int       `abi:"lltv"`
}

func (m *MarketParams) ToMarketContractParams() *MarketContractParams {
	return &MarketContractParams{
		LoanToken:       m.LoanToken,
		CollateralToken: m.CollateralToken,
		Oracle:          m.Oracle,
		Irm:             m.Irm,
		LLTV:            m.LLTV,
	}
}

func (m *MarketParams) IsETHCorrelated() bool {
	return strings.Contains(m.CollateralTokenStr, "ETH") && strings.Contains(m.LoanTokenStr, "ETH")
}

func (m *MarketParams) GetPair() string {
	return fmt.Sprintf("%s/%s", m.CollateralTokenStr, m.LoanTokenStr)
}

func (m *MarketParams) GetCollateralToken() common.Address {
	return m.CollateralToken
}

func (m *MarketParams) GetLoanToken() common.Address {
	return m.LoanToken
}

func (m *MarketParams) GetLLTV() *big.Int {
	return m.LLTV
}

func (m *MarketParams) MaxSlippage() float64 {
	lltvF, _ := new(big.Float).SetInt(m.LLTV).Float64()
	lltvPct := lltvF / 1e18 * 100
	bonus := 100 - lltvPct
	const gasCushion = 0.1
	return bonus - gasCushion
}
