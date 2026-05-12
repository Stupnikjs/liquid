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

func NewConsumer(infra *lqtypes.Infra, store *lqtypes.Store, logger chan string, ch <-chan cache.BorrowPosition) *Consumer {
	return &Consumer{
		Infra:  infra,
		Store:  store,
		Logger: logger,
		Ch:     ch,
	}
}

type Consumer struct {
	Infra  *lqtypes.Infra
	Store  *lqtypes.Store
	Logger chan string
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
			snap := c.Store.GetSnapshot(pos.MarketID)
			if snap == nil {
				out.SimErr = fmt.Errorf("snap nil")
				return out
			}
			c.liquidateWrapper(ctx, snap, &pos)
		}
	}
}

func (c *Consumer) liquidateWrapper(ctx context.Context, snap market.Snapshot, p *cache.BorrowPosition) {
	result := c.SimulateAndPreComputeTx(ctx, snap, p)
	if result.SimErr != nil {
		c.log(fmt.Sprintf("[liq] simulation failed for %s: %v", p.Address, result.SimErr))
		return
	}
	if !result.IsLiquidable {
		c.log("[liq] not profitable")
		return
	}
	market := c.Store.MarketMap[p.MarketID]
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
	_, err = onchain.SendSignedTx(ctx, c.Infra.Conn, c.Infra.Config.Addresses.Wallet, c.Infra.Config.Signer, tx)
	return err
}
