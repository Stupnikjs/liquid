package utils

import (
	"context"
	"time"
)

// runTicker est un worker générique réutilisable

func RunTicker(ctx context.Context, interval time.Duration, errCh chan<- error, fn func() error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := fn(); err != nil {
				select {
				case errCh <- err:
				default: // évite le blocage si errCh est plein
				}
			}
		}
	}
}
