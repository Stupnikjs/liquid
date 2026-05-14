package lqtypes

import (
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/internal/connector"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
)

type Infra struct {
	Conn   *connector.Connector
	Config config.Config
}

type Store struct {
	MarketReader MarketReader
	MarketMap    map[[32]byte]morpho.MarketParams
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

type SingleQuoterFunc func(
	conn *connector.Connector,
	marketp morpho.MarketParams,
	routerQuoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
	fee uint32,
) (PoolEdge, bool)

type QuoteParams struct {
	Fn     SingleQuoterFunc
	Router common.Address
}

type LiquidateArgs struct {
	MarketParams morpho.MarketContractParams
	Borrower     common.Address
	SeizedAssets *big.Int
	RepaidShares *big.Int
	SwapRouter   common.Address
	PoolFee      *big.Int
	MinOut       *big.Int
}

// encode liquidate args after selector
func (args LiquidateArgs) EncodeLiquidateCalldata() ([]byte, error) {
	return config.FuncLiquidate.EncodeArgs(
		args.MarketParams,
		args.Borrower,
		args.SeizedAssets,
		args.RepaidShares,
		args.SwapRouter,
		args.PoolFee,
		args.MinOut,
	)
}
