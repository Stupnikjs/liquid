package connector

import (
	"fmt"
	"log"
	"time"

	"github.com/lmittmann/w3"
)

func (c *Connector) reconnectWS() {
	backoff := 1 * time.Second
	for {
		client, err := w3.Dial(c.WsRPC)
		if err != nil {
			fmt.Printf("ws reconnect failed: %v, retry in %s\n", err, backoff)
			time.Sleep(backoff)
			backoff = min(backoff*2, 30*time.Second)
			continue
		}

		c.mu.Lock()
		old := c.ClientWS
		c.ClientWS = client
		c.mu.Unlock()

		if old != nil {
			old.Close()
		}
		log.Printf("[connector] WS reconnected to %s", c.WsRPC)
		return
	}
}
