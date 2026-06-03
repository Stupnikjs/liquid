package liquidate

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/internal/onchain"
	"github.com/Stupnikjs/liquid/internal/utils"
	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/Stupnikjs/liquid/pkg/swap"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Liquidable struct {
	Pos          *cache.BorrowPosition
	RepayShares  *big.Int
	SeizeAssets  *big.Int
	MinOut       *big.Int
	EstProfit    *big.Int
	GasEstimate  uint64
	SimulatedAt  time.Time
	IsLiquidable bool
	CallData     []byte
}

func NewConsumer(conn connector.Connector, config lqtypes.Config, cache *cache.Cache, swapCache *swap.RouteCache, ch chan cache.BorrowPosition) *Consumer {
	return &Consumer{
		Conn:      conn,
		Config:    config,
		SwapCache: swapCache,
		Cache:     cache,
		Ch:        ch,
	}
}

type Consumer struct {
	Conn      connector.Connector
	Config    lqtypes.Config
	SwapCache *swap.RouteCache
	Cache     *cache.Cache
	Ch        chan cache.BorrowPosition
}

func (c *Consumer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case pos := <-c.Ch:
			m := c.Cache.MarketMap[pos.MarketID]
			snap := c.Cache.Markets.GetSnapshot(pos.MarketID)
			if snap == nil {
				continue
			}
			c.liquidateWrapper(ctx, m, snap, &pos)
		}
	}
}

func (c *Consumer) liquidateWrapper(ctx context.Context, market morpho.MarketParams, snap *cache.MarketSnapshot, p *cache.BorrowPosition) {

	result, err := c.SimulateAndPreComputeTx(ctx, market, snap, p)
	if err != nil {
		log.Printf("[liq] simulation failed for %s: %v", p.Address, err)
		return
	}

	log.Printf("[liq] sending tx for %s seized=%s market %s  \n", p.Address, utils.FormatDecimals(result.SeizeAssets, int(market.CollateralTokenDecimals)), market.GetPair())
	err = c.LiquidateCall(ctx, result.CallData, result.GasEstimate)
	if err != nil {
		log.Printf("[liq] tx failed for %s: %v", p.Address, err)
		return
	}
	log.Printf("[liq] ✓ liquidated pair%s marketid:%s borrower:%s", market.GetPair(), hexutil.Encode(market.ID[:]), p.Address)
}

func (c *Consumer) LiquidateCall(ctx context.Context, calldata []byte, gasEstimate uint64) error {
	tx := onchain.TxParams{
		To:          &c.Config.Addresses.LiquidatorContract,
		Calldata:    calldata,
		GasEstimate: gasEstimate,
	}
	_, err := onchain.SendSignedTx(ctx, c.Conn, c.Config.Addresses.Wallet, c.Config.Signer, tx)
	return err
}
