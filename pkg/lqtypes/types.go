package lqtypes

import (
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/pkg/config"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common"
)

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

type LiquidateArgs struct {
	MarketParams morpho.MarketContractParams
	Borrower     common.Address
	SeizedAssets *big.Int
	RepaidShares *big.Int
	SwapRouter   common.Address
	PoolFee      *big.Int
	MinOut       *big.Int
}

func EncodeLiquidateCalldata(args LiquidateArgs) ([]byte, error) {
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
