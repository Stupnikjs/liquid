package morpho

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/ethereum/go-ethereum/common"
)

var IRM = common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687")

type ChainConfig struct {
	WalletAddress        common.Address
	LiquidatorAddress    common.Address
	UniswapRouterAddress common.Address
	Signer               *config.Signer
	Name                 string
}

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
		Irm:             m.Irm,
		Lltv:            m.LLTV,
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
