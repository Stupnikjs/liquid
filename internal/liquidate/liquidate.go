package liquidate

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/internal/onchain"
	"github.com/Stupnikjs/liquid/internal/swap"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/lmittmann/w3/w3types"
)

// mockable dans testutil
type EthCaller interface {
	EthCallCtx(ctx context.Context, calls []w3types.RPCCaller) error
}

type Liquidable struct {
	Pos          *cache.BorrowPosition
	RepayShares  *big.Int
	SeizeAssets  *big.Int
	MinOut       *big.Int
	EstProfit    *big.Int
	GasEstimate  uint64
	SimulatedAt  time.Time
	SimErr       error
	IsLiquidable bool
	CallData     []byte
}

func NewConsumer(infra *lqtypes.Infra, cache *cache.Cache, swapCache *swap.RouteCache, logger chan string, ch <-chan cache.BorrowPosition) *Consumer {
	return &Consumer{
		Infra:     infra,
		Logger:    logger,
		SwapCache: swapCache,
		Cache:     cache,
		Ch:        ch,
	}
}

type Consumer struct {
	Infra     *lqtypes.Infra
	Logger    chan string
	SwapCache *swap.RouteCache
	Cache     *cache.Cache
	Ch        <-chan cache.BorrowPosition
}

func (c *Consumer) log(msg string) {
	select {
	case c.Logger <- msg:
	default: // drop if full — never block the liquidation path
	}
}

func (c *Consumer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case pos := <-c.Ch:
			m := c.Cache.MarketMap[pos.MarketID]
			snap := c.Cache.Markets.GetSnapshot(pos.MarketID)
			c.liquidateWrapper(ctx, m, snap, &pos)
		}
	}
}

func (c *Consumer) liquidateWrapper(ctx context.Context, market morpho.MarketParams, snap *cache.MarketSnapshot, p *cache.BorrowPosition) {
	route, ok := c.SwapCache.GetRoute(market.CollateralToken, market.LoanToken)
	if !ok {
		c.log("no route found in cache for these tokens")
		return
	}
	fee := route.Hops[0].Fee // change for multihop
	result := c.SimulateAndPreComputeTx(ctx, market, snap, int64(fee), p)
	if result.SimErr != nil {
		c.log(fmt.Sprintf("[liq] simulation failed for %s: %v", p.Address, result.SimErr))
		return
	}
	if !result.IsLiquidable {
		c.log("[liq] not profitable")
		return
	}
	c.log(fmt.Sprintf("[liq] sending tx for %s seized=%s market %s ", p.Address, utils.FormatDecimals(result.SeizeAssets, int(market.CollateralTokenDecimals)), market.GetPair()))
	err := c.LiquidateCall(ctx, result.CallData, result.GasEstimate)
	if err != nil {
		c.log(fmt.Sprintf("[liq] tx failed for %s: %v", p.Address, err))
		return
	}
	c.log(fmt.Sprintf("[liq] ✓ liquidated pair%s marketid:%s borrower:%s", market.GetPair(), hexutil.Encode(market.ID[:]), p.Address))
}

func (c *Consumer) LiquidateCall(ctx context.Context, calldata []byte, gasEstimate uint64) error {
	tx := onchain.TxParams{
		To:          &c.Infra.Config.Addresses.LiquidatorContract,
		Calldata:    calldata,
		GasEstimate: gasEstimate,
	}
	_, err := onchain.SendSignedTx(ctx, c.Infra.Conn, c.Infra.Config.Addresses.Wallet, c.Infra.Config.Signer, tx)
	return err
}

// encode liquidate args after selector
func EncodeLiquidateCalldata(args lqtypes.LiquidateArgs) ([]byte, error) {
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
