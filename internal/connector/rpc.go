package connector

import (
	"fmt"
	"log"
	"time"

	"github.com/lmittmann/w3"
)

func (c *Connector) SwapHTPP() {
	backoff := 1 * time.Second
	for i := range c.HTTP {
		endpoint := c.HTTP[(c.currHTTP+i)%len(c.HTTP)]
		client, err := w3.Dial(endpoint)
		if err != nil {
			fmt.Printf("http swap failed for %s: %v, retry in %s\n", endpoint, err, backoff)
			time.Sleep(backoff)
			backoff = min(backoff*2, 30*time.Second)
			continue
		}

		c.mu.Lock()
		old := c.ClientHTTP
		c.ClientHTTP = client
		c.currHTTP = (c.currHTTP + i) % len(c.HTTP)
		c.mu.Unlock()

		if old != nil {
			old.Close()
		}
		log.Printf("[connector] HTTP switched to %s", endpoint)
		return
	}
	// All endpoints failed — keep retrying from start after delay
	time.Sleep(backoff)
	c.SwapHTPP()
}
func (c *Connector) reconnectWS() {
	backoff := 1 * time.Second
	for i := range c.WS {
		endpoint := c.WS[(c.currWS+i)%len(c.WS)]
		client, err := w3.Dial(endpoint)
		if err != nil {
			fmt.Printf("ws reconnect failed for %s: %v, retry in %s\n", endpoint, err, backoff)
			time.Sleep(backoff)
			backoff = min(backoff*2, 30*time.Second)
			continue
		}

		c.mu.Lock()
		old := c.ClientWS
		c.ClientWS = client
		c.currWS = (c.currWS + i) % len(c.WS)
		c.mu.Unlock()

		if old != nil {
			old.Close()
		}
		log.Printf("[connector] WS reconnected to %s", endpoint)
		return
	}
	// All endpoints failed — keep retrying from start after delay
	time.Sleep(backoff)
	c.reconnectWS()
}
