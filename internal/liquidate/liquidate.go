package liquidate

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/internal/onchain"
	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/Stupnikjs/liquid/pkg/swap"
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

func NewLiquidateConsumer(conn connector.Connector, config config.Config, cache cache.MarketCache, swapCache *swap.RouteCache, ch chan cache.BorrowPosition) *LiquidateConsumer {
	return &LiquidateConsumer{
		Conn:      conn,
		Config:    config,
		SwapCache: swapCache,
		Cache:     cache,
		Ch:        ch,
	}
}

type LiquidateConsumer struct {
	Conn      connector.Connector
	Config    config.Config
	SwapCache *swap.RouteCache
	Cache     cache.MarketCache
	Ch        chan cache.BorrowPosition
}

func (c *LiquidateConsumer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case pos := <-c.Ch:
			m := c.Cache.MorphoMarketByID(pos.MarketID)
			snap := c.Cache.GetSnapshot(pos.MarketID)
			if snap == nil {
				continue
			}
			c.liquidateWrapper(ctx, m, snap, &pos)
		}
	}
}

// args snap market can be deleted since all is in consumer only need pos
func (c *LiquidateConsumer) liquidateWrapper(ctx context.Context, market morpho.MarketParams, snap *cache.MarketSnapshot, p *cache.BorrowPosition) {

	result, err := c.SimulateAndPreComputeTx(ctx, market, snap, p)
	if err != nil {
		log.Printf("[liq] simulation failed for %s: %v", p.Address, err)
		return
	}

	log.Printf("sending liquidation call %s \n", p.String(market, snap.Oracle.Price))
	err = c.LiquidateCall(ctx, result.CallData, result.GasEstimate)
	if err != nil {
		log.Printf("[liq] tx failed for %s: %v", p.Address, err)
		return
	}

}

func (c *LiquidateConsumer) LiquidateCall(ctx context.Context, calldata []byte, gasEstimate uint64) error {
	tx := onchain.TxParams{
		To:          &c.Config.Addresses.LiquidatorContract,
		Calldata:    calldata,
		GasEstimate: gasEstimate,
	}

	_, err := onchain.SendSignedTx(ctx, c.Conn, c.Config.Addresses.Wallet, c.Config.Signer, tx)
	return err
}
