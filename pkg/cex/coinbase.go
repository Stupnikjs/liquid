package cex

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/coder/websocket"
)

const (
	coinbaseWSURL = "wss://ws-feed.exchange.coinbase.com"
	reconnectWait = 2 * time.Second
	maxBackoff    = 30 * time.Second
)

var subscribeMsg = map[string]any{
	"type":        "subscribe",
	"product_ids": []string{"BTC-USD", "ETH-USD", "XRP-USD", "BTC-EUR"},
	"channels":    []string{"ticker"},
}

type coinbaseTicker struct {
	Type      string `json:"type"`
	ProductID string `json:"product_id"`
	BestBid   string `json:"best_bid"`
	BestAsk   string `json:"best_ask"`
}

type AssetPrice struct {
	MidPrice float64
	Ts       int64
}

type PriceUpdate struct {
	ProductID string
	Price     AssetPrice
}

type CoinbaseConnector struct {
	PriceUpdatesCh chan PriceUpdate // plus de pointeur, PriceUpdate est petit
}

func NewCoinbaseConnector() *CoinbaseConnector {
	return &CoinbaseConnector{
		PriceUpdatesCh: make(chan PriceUpdate, 1),
	}
}

func (connector *CoinbaseConnector) Run(ctx context.Context) {
	backoff := reconnectWait
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := connector.connect(ctx); err != nil {
				log.Printf("[coinbase] disconnected: %v — retry in %s", err, backoff)
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

func (connector *CoinbaseConnector) connect(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, coinbaseWSURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.CloseNow()

	subscribePayload, _ := json.Marshal(subscribeMsg)
	if err := conn.Write(ctx, websocket.MessageText, subscribePayload); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	log.Println("[coinbase] connected")

	for {
		_, rawMessage, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				return nil
			}
			return fmt.Errorf("read: %w", err)
		}

		var ticker coinbaseTicker
		if err := json.Unmarshal(rawMessage, &ticker); err != nil {
			continue
		}

		if ticker.Type != "ticker" {
			continue
		}

		// calcul du mid-price directement ici
		bid, err := strconv.ParseFloat(ticker.BestBid, 64)
		if err != nil {
			continue
		}
		ask, err := strconv.ParseFloat(ticker.BestAsk, 64)
		if err != nil {
			continue
		}

		update := PriceUpdate{
			ProductID: ticker.ProductID,
			Price: AssetPrice{
				MidPrice: (bid + ask) / 2,
				Ts:       time.Now().Unix(),
			},
		}

		select {
		case connector.PriceUpdatesCh <- update:
		default:
		}
	}
}

func (connector *CoinbaseConnector) PriceCh() <-chan PriceUpdate {
	return connector.PriceUpdatesCh
}
