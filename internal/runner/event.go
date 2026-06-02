package runner

import (
	"context"
	"log"

	"github.com/Stupnikjs/liquid/internal/config/abi"
	"github.com/Stupnikjs/liquid/internal/onchain"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

func (r *Runner) SubscribePositionRoutine(ctx context.Context) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{r.Infra.Config.Addresses.Morpho},
		Topics: [][]common.Hash{{
			abi.EventBorrow.Topic0,
			abi.EventRepay.Topic0,
			abi.EventSupplyCollateral.Topic0,
			abi.EventLiquidate.Topic0,
			abi.EventAccrueInterest.Topic0,
		}},
	}
	ch, err := r.Infra.Conn.SubscribeLogs(ctx, query)
	r.EventCh = ch
	if err != nil {
		log.Printf("Error subscribing to logs: %v", err)
	}

}

func (r *Runner) EventListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-r.EventCh:
			if !ok {
				return
			}
			onchain.ProcessEvents(r.Cache.Markets, event)
		}
	}
}
