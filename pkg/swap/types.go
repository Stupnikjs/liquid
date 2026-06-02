package swap

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3/w3types"
)

type RPCClient interface {
	CallCtx(ctx context.Context, calls ...w3types.RPCCaller) error
	Close() error
}

type Dex interface {
	// Quote returns the best PoolEdge for a given amountIn.
	// Returns false if no route satisfies the market's slippage constraint.
	BestAmountIn(conn RPCClient, marketp MorphoMarket, amountIn *big.Int, oraclePrice *big.Int, rateLimit time.Duration) (PoolEdge, bool)
	// DEX returns a human-readable identifier for logging and metrics.
	DEX() string
	QuoterAddress() common.Address
	RouterAddress() common.Address
}

type MorphoMarket interface {
	GetPair() string
	GetCollateralToken() common.Address
	GetLoanToken() common.Address
	GetLLTV() *big.Int
	MaxSlippage() float64
}

// mapping to liquidate contract struct
type SwapStep struct {
	Target         common.Address
	Data           []byte
	TokenIn        common.Address
	TokenOut       common.Address
	AmountInOffset *big.Int
}

type PoolEdge struct {
	TokenIn        common.Address
	TokenOut       common.Address
	Quoter         common.Address // useless i think
	Router         common.Address
	Fee            uint32
	WCSlippage     float64
	WCAmountIn     *big.Int
	WCAmountOut    *big.Int
	CalibratedAt   time.Time
	DexName        string
	AmountInOffset int64
	PriceAtQuote   *big.Int
}

type QuoterFunc func(
	conn RPCClient,
	marketp MorphoMarket,
	quoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
	rateLimit time.Duration,
) (PoolEdge, bool)

type QuotSingleFunc func(conn RPCClient,
	marketp MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	fee uint32,
) (PoolEdge, bool)
