package cex

const (
	krakenWSURL = "wss://ws.kraken.com/v2"
)

var krakenSubscribeMsg = map[string]any{
	"method": "subscribe",
	"params": map[string]any{
		"channel": "ticker",
		"symbol":  []string{"BTC/USD", "ETH/USD"},
	},
}

type krakenTickerData struct {
	Symbol string  `json:"symbol"`
	Bid    float64 `json:"bid"`
	Ask    float64 `json:"ask"`
}

type krakenMessage struct {
	Channel string             `json:"channel"`
	Type    string             `json:"type"`
	Data    []krakenTickerData `json:"data"`
}

/*
// ── KRAKEN CONNECTOR ─────────────────────────────────────────────────────────

type KrakenConnector struct {
	PriceUpdatesCh chan PriceUpdate
}

func NewKrakenConnector() *KrakenConnector {
	return &KrakenConnector{
		PriceUpdatesCh: make(chan PriceUpdate, 1),
	}
}

func (k *KrakenConnector) Run(ctx context.Context) {
	backoff := reconnectWait
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := k.connect(ctx); err != nil {
				log.Printf("[kraken] disconnected: %v — retry in %s", err, backoff)
				select {
				case <-time.After(backoff):
					backoff = min(backoff*2, maxBackoff)
				case <-ctx.Done():
					return
				}
			} else {
				backoff = reconnectWait
			}
		}
	}
}

func (k *KrakenConnector) connect(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, krakenWSURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.CloseNow()

	payload, _ := json.Marshal(krakenSubscribeMsg)
	if err := conn.Write(ctx, websocket.MessageText, payload); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	log.Println("[kraken] connected")

	for {
		_, raw, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				return nil
			}
			return fmt.Errorf("read: %w", err)
		}

		var msg krakenMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		if msg.Channel != "ticker" || (msg.Type != "update" && msg.Type != "snapshot") {
			continue
		}

		for _, d := range msg.Data {
			var productID string
			switch d.Symbol {
			case "BTC/USD":
				productID = "BTC-USD"
			case "ETH/USD":
				productID = "ETH-USD"
			default:
				continue
			}

			update := PriceUpdate{
				ProductID: productID,
				Price: AssetPrice{
					MidPrice: (d.Bid + d.Ask) / 2,
					Ts:       time.Now().Unix(),
				},
			}

			select {
			case k.PriceUpdatesCh <- update:
			default:
			}
		}
	}
}

func (k *KrakenConnector) PriceCh() <-chan PriceUpdate {
	return k.PriceUpdatesCh
}

// ── AGGREGATOR ───────────────────────────────────────────────────────────────

type CexAggregator struct {
	coinbase *CoinbaseConnector
	kraken   *KrakenConnector
	outCh    chan CexCache

	lastCoinbaseBTC AssetPrice
	lastCoinbaseETH AssetPrice
	coinbaseAlive   bool
}

func NewCexAggregator() *CexAggregator {
	return &CexAggregator{
		coinbase:      NewCoinbaseConnector(),
		kraken:        NewKrakenConnector(),
		outCh:         make(chan CexCache, 1),
		coinbaseAlive: true,
	}
}

func (a *CexAggregator) Run(ctx context.Context) {
	go a.coinbase.Run(ctx)
	go a.kraken.Run(ctx)

	coinbaseCh := a.coinbase.PriceCh()
	krakenCh := a.kraken.PriceCh()

	coinbaseTimeout := time.NewTicker(10 * time.Second)
	defer coinbaseTimeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case update := <-coinbaseCh:
			a.applyUpdate(&a.lastCoinbaseBTC, &a.lastCoinbaseETH, update)
			a.coinbaseAlive = true
			coinbaseTimeout.Reset(10 * time.Second)
			a.emit()

		case update := <-krakenCh:
			if !a.coinbaseAlive {
				log.Println("[aggregator] Coinbase stale, using Kraken")
				a.applyUpdate(&a.lastCoinbaseBTC, &a.lastCoinbaseETH, update)
				a.emit()
			}

		case <-coinbaseTimeout.C:
			if a.coinbaseAlive {
				log.Println("[aggregator] Coinbase timeout — fallback Kraken")
				a.coinbaseAlive = false
			}
		}
	}
}

func (a *CexAggregator) applyUpdate(btc, eth *AssetPrice, update PriceUpdate) {
	switch update.ProductID {
	case "BTC-USD":
		*btc = update.Price
	case "ETH-USD":
		*eth = update.Price
	}
}

func (a *CexAggregator) PriceCh() <-chan CexCache {
	return a.outCh
}
*/
