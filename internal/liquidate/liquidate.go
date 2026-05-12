package liquidate

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/internal/onchain"
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

func NewConsumer(infra *lqtypes.Infra, store lqtypes.Store, logger chan string, ch <-chan cache.BorrowPosition) *Consumer {
	return &Consumer{
		Infra:  infra,
		Logger: logger,
		Store:  store,
		Ch:     ch,
	}
}

type Consumer struct {
	Infra  *lqtypes.Infra
	Logger chan string
	Store  lqtypes.Store
	Ch     <-chan cache.BorrowPosition
}

func (c *Consumer) log(msg string) {
	select {
	case c.Logger <- msg:
	default: // drop if full — never block the liquidation path
	}
}

// Pure math, zero RPC — unit testable

// ABI encode — testable isolément

func (c *Consumer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case pos := <-c.Ch:
			snap := c.Store.MarketReader.GetSnapshot(pos.MarketID)
			m := c.Store.MarketMap[pos.MarketID]
			if snap == nil {
				c.log(fmt.Sprintf("[liq] no snapshot for market %s", pos.MarketID))
				continue
			}
			c.liquidateWrapper(ctx, *snap, m, &pos)
		}
	}
}

func (c *Consumer) liquidateWrapper(ctx context.Context, snap cache.MarketSnapshot, market morpho.MarketParams, p *cache.BorrowPosition) {
	result := c.SimulateAndPreComputeTx(ctx, market, int64(snap.Stats.SwapFee), p)
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
