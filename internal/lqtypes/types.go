package lqtypes

import (
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/config"
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

type MarketReader interface {
	Ids() [][32]byte
	GetSnapshot(id [32]byte) *cache.MarketSnapshot
	Update(id [32]byte, fn func(m *cache.Market))
}

// encode liquidate args after selector
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
