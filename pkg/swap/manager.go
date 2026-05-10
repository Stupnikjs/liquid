package swap

import (
	"math/big"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/internal/connector"
	"github.com/Stupnikjs/liquid/pkg/config"
	"github.com/Stupnikjs/liquid/pkg/lqtypes"
	"github.com/Stupnikjs/liquid/pkg/morpho"
)

type Consumer struct {
	Conn      *connector.Connector
	Config    config.Config
	Cache     cache.MarketReader
	MarketMap map[[32]byte]morpho.MarketParams
	Logger    chan string
}

func NewConsumer(conn *connector.Connector, mReader cache.MarketReader, marketMap map[[32]byte]morpho.MarketParams, conf config.Config, logchan chan string) *Consumer {
	return &Consumer{
		Conn:      conn,
		Config:    conf,
		MarketMap: marketMap,
		Logger:    logchan,
	}
}

func (c *Consumer) SingleHop(MaxCollateralPos, OraclePrice *big.Int) map[[32]byte]lqtypes.PoolEdge {
	swapMap := make(map[[32]byte]lqtypes.PoolEdge, len(c.MarketMap))
	for _, morphoM := range c.MarketMap {

		result, found := QuoteBinarySearch(c.Conn, morphoM, c.Config.Addresses.UniSwapQuoter, MaxCollateralPos, OraclePrice)
		if found {
			swapMap[morphoM.ID] = result
		}

	}
	return swapMap
}

func (c *Consumer) MultiHop(MaxCollateralPos, OraclePrice *big.Int) map[[32]byte][][]lqtypes.PoolEdge {
	returnMap := make(map[[32]byte][][]lqtypes.PoolEdge, len(c.MarketMap))
	swapMap := c.SingleHop(MaxCollateralPos, OraclePrice)
	graph := NewPoolGraph()
	for _, v := range swapMap {
		graph.AddPool(v)
	}
	for id, m := range c.MarketMap {
		route := graph.FindRoutes(m.CollateralToken, m.LoanToken, 1)
		returnMap[id] = route
	}
	return returnMap
}
