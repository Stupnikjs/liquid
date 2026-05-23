package lqtypes

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

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
	Quoter         common.Address
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

type Dex struct {
	QuoterAddr common.Address
	RouterAddr common.Address
	Quoter     QuoterFunc
	Name       string
}

type QuoterFunc func(
	conn EthCaller,
	marketp MorphoMarket,
	quoterAddr common.Address,
	amountIn, oraclePrice *big.Int,
	rateLimit time.Duration,
) (PoolEdge, bool)

type QuotSingleFunc func(conn EthCaller,
	marketp MorphoMarket,
	quoterAddr common.Address,
	amountIn *big.Int,
	oraclePrice *big.Int,
	fee uint32,
) (PoolEdge, bool)

func (d *Dex) Quote(conn EthCaller, marketp MorphoMarket, amountIn, oraclePrice *big.Int,
	rateLimit time.Duration) (PoolEdge, bool) {
	return d.Quoter(conn, marketp, d.QuoterAddr, amountIn, oraclePrice, rateLimit)
}
