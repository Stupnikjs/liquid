package runner

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/Stupnikjs/morpho-sepolia/internal/cache"
	"github.com/Stupnikjs/morpho-sepolia/internal/connector"
	"github.com/Stupnikjs/morpho-sepolia/internal/logging"
	"github.com/Stupnikjs/morpho-sepolia/internal/onchain"
	"github.com/Stupnikjs/morpho-sepolia/pkg/config"
	"github.com/Stupnikjs/morpho-sepolia/pkg/swap"
)

type Runner struct {
	Cache       *cache.Cache
	Conn        *connector.Connector
	Logger      chan string
	Config      config.Config
	LiquidateCh chan cache.BorrowPosition
	// Config avec signer
}

func NewRunner(initedCache *cache.Cache, conf config.Config, logfile string) *Runner {
	conn := connector.NewConnector(conf.RPC.HTTP, conf.RPC.WS)
	logger := logging.NewLogger(context.Background(), logfile)
	return &Runner{
		Cache:       initedCache,
		Conn:        conn,
		Logger:      logger,
		Config:      conf,
		LiquidateCh: make(chan cache.BorrowPosition, 1),
	}
}

func formatAmount(amount *big.Int, decimals uint8) string {
	if amount == nil {
		return "0"
	}
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	amountF := new(big.Float).SetInt(amount)
	result := new(big.Float).Quo(amountF, divisor)
	return result.Text('f', 4)
}

func (r *Runner) Init(ctx context.Context) {
	fmt.Println("INITING __ ")
	err := r.ApiCallRoutine(ctx)
	if err != nil {
		fmt.Println(err)
	}
	r.OnChainRefreshAll()
	fmt.Println("all pos", r.Cache.Markets.AllPosLen())
	// CHECK Liquidity on markets with uniswap
	r.Cache.Markets.Range(func(id [32]byte) {
		snap := r.Cache.Markets.GetSnapshot(id)
		if snap == nil {
			fmt.Println("SNAP IS NIL __ OK ")
			return
		}
		fmt.Println("SNAP ISNT NIL __ OK ")
		morphoM := r.Cache.MarketMap[id]
		result, err := swap.Quote(r.Conn.ClientHTTP, morphoM, r.Config.Addresses.UniSwapQuoter, snap.Stats.MaxCollateralPos, snap.Oracle.Price)
		if err != nil {
			fmt.Println("QUOTE __ IS NIL ", err)
			r.Cache.Markets.Update(id, func(m *cache.Market) {
				m.Canceled = true
			})
			return
		}
		fmt.Printf("Pair %s/%s | source: uniswap | slippage: %f%% | amountIn: %s %s\n",
			morphoM.CollateralTokenStr,
			morphoM.LoanTokenStr,
			result.Slippage,
			formatAmount(result.AmountIn, uint8(morphoM.CollateralTokenDecimals)),
			morphoM.CollateralTokenStr,
		)
		r.Cache.Markets.Update(id, func(m *cache.Market) {
			m.Stats.MaxUniSwappable = result.AmountIn
			m.Stats.SwapFee = result.Fee
		})

	})

	fmt.Println(len(r.Cache.Markets.Ids()))
}

func (r *Runner) OnChainRefreshAll() {
	var wg sync.WaitGroup
	for _, id := range r.Cache.Markets.Ids() {
		wg.Add(1)
		go func(id [32]byte) {
			defer wg.Done()
			onchain.OnChainRefresh(r.Conn, r.Cache.Markets, r.Cache.GetMorphoMarketFromId(id), id, r.Config.Addresses.Morpho)
		}(id)
	}
	wg.Wait()
}
