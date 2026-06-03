package runner

import (
	"context"
	"log"

	"github.com/Stupnikjs/liquid/internal/config/abi"
	"github.com/Stupnikjs/liquid/internal/onchain"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

func (r *MarketConsumer) SubscribePositionRoutine(ctx context.Context) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{r.Config.Addresses.Morpho},
		Topics: [][]common.Hash{{
			abi.EventBorrow.Topic0,
			abi.EventRepay.Topic0,
			abi.EventSupplyCollateral.Topic0,
			abi.EventLiquidate.Topic0,
			abi.EventAccrueInterest.Topic0,
		}},
	}
	ch, err := r.Conn.SubscribeLogs(ctx, query)
	r.EventCh = ch
	if err != nil {
		log.Printf("Error subscribing to logs: %v", err)
	}

}

func (m *MarketConsumer) EventListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-m.EventCh:
			if !ok {
				return
			}
			onchain.ProcessEvents(m.Cache.Markets, event)
		}
	}
}
