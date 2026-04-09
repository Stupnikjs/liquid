/*
package logging

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/Stupnikjs/morpho-sepolia/internal/utils"
)

func WriteLogRoutine(ctx context.Context, filename string) {
	var mu sync.Mutex
	logCache := make(map[int64]string)

	pathLog := path.Join("logs", filename)
	file, _ := os.Create(pathLog)
	defer file.Close()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-logChannel:
				mu.Lock()
				logCache[time.Now().Unix()] = msg
				mu.Unlock()

			case err := <-errCh:
				mu.Lock()
				logCache[time.Now().Unix()] = err.Error()
				mu.Unlock()
			}
		}
	}()

	utils.RunTicker(ctx, 2*time.Minute, errCh, func() error {
		mu.Lock()
		defer mu.Unlock()
		if len(logCache) == 0 {
			return nil
		}

		for ts, msg := range logCache {
			_, _ = fmt.Fprintf(file, "[%s] %s\n", time.Unix(ts, 0).Format(time.RFC3339), msg)

		}

		// vide le cache
		clear(logCache)
		return nil
	})
}
*/