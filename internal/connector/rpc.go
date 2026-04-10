package connector

import (
	"fmt"
	"log"
	"time"

	"github.com/lmittmann/w3"
)

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

// SwapToMainHttp tries the main index first, falls back to rotating through others.
func (c *Connector) SwapToMainHttp() (bool, error) {
	client, err := w3.Dial(c.HTTP[c.MainIndex])

	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		c.currHTTP = (c.currHTTP + 1) % len(c.HTTP)
		fallback, err2 := w3.Dial(c.HTTP[c.currHTTP])
		if err2 != nil {
			return false, fmt.Errorf("swap to main failed, fallback also failed: %w", err2)
		}
		if c.ClientHTTP != nil {
			c.ClientHTTP.Close()
		}
		c.ClientHTTP = fallback
		return false, fmt.Errorf("swap to main failed, using fallback: %w", err)
	}

	if c.ClientHTTP != nil {
		c.ClientHTTP.Close()
	}
	c.ClientHTTP = client
	c.currHTTP = c.MainIndex
	return true, nil
}

func (c *Connector) RefreshRPC() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.currHTTP = (c.currHTTP + 1) % len(c.HTTP)
	client, err := w3.Dial(c.HTTP[c.currHTTP])
	if err != nil {
		return fmt.Errorf("RefreshRPC: failed to dial %s: %w", c.HTTP[c.currHTTP], err)
	}
	if c.ClientHTTP != nil {
		c.ClientHTTP.Close()
	}
	c.ClientHTTP = client
	return nil
}
