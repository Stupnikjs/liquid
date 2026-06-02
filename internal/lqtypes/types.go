package lqtypes

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Types with no internal dep

type Infra struct {
	Conn   connector.Connector
	Config Config
}

type Addresses struct {
	LiquidatorContract common.Address
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
	Endpoints connector.RPCEndpoints
	Dexs      []Dex
}

type MarketContractParams struct {
	LoanToken       common.Address
	CollateralToken common.Address
	Oracle          common.Address
	Irm             common.Address
	Lltv            *big.Int
}
