package lqtypes

import (
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/swap"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Types with no internal dep

type Infra struct {
	Conn   EthCaller
	Config Config
}

type Addresses struct {
	LiquidatorContract common.Address
	UniSwapRouter      common.Address
	UniSwapQuoter      common.Address
	Wallet             common.Address
	Morpho             common.Address
}
type Signer struct {
	key    *ecdsa.PrivateKey
	signer types.Signer
}

type QuoteExactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	AmountIn          *big.Int
	Fee               *big.Int
	SqrtPriceLimitX96 *big.Int
}

type Config struct {
	Signer    *Signer
	Addresses Addresses
	ChainID   uint32
	RPC       struct {
		HTTP []string
		WS   []string
	}
	Dexs []swap.Dex
}

type PoolEdge struct {
	TokenIn      common.Address
	TokenOut     common.Address
	Router       common.Address
	Fee          uint32
	WCSlippage   float64
	WCAmountIn   *big.Int
	WCAmountOut  *big.Int
	CalibratedAt time.Time
}

type MarketContractParams struct {
	LoanToken       common.Address
	CollateralToken common.Address
	Oracle          common.Address
	Irm             common.Address
	Lltv            *big.Int
}

type LiquidateArgs struct {
	MarketParams MarketContractParams
	Borrower     common.Address
	SeizedAssets *big.Int
	RepaidShares *big.Int
	SwapRouter   common.Address
	PoolFee      *big.Int
	MinOut       *big.Int
}
