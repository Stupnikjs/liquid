package swap

import (
	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/internal/connector"
	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/pkg/morpho"
)

type Consumer struct {
	Conn      *connector.Connector
	Config    config.Config
	Cache     lqtypes.MarketReader
	MarketMap map[[32]byte]morpho.MarketParams
	Logger    chan string
}

func NewConsumer(conn *connector.Connector, mReader lqtypes.MarketReader, marketMap map[[32]byte]morpho.MarketParams, conf config.Config, logchan chan string) *Consumer {
	return &Consumer{
		Conn:      conn,
		Config:    conf,
		MarketMap: marketMap,
		Logger:    logchan,
	}
}

/*
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
*/
