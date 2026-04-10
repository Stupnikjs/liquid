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

func NewLogger(ctx context.Context, filename string) chan string {
	var mu sync.Mutex
	logChannel := make(chan string, 100) // ✅ buffered pour éviter les blocages
	logCache := make(map[int64]string)

	pathLog := path.Join("logs", filename)
	file, err := os.Create(pathLog)
	if err != nil {
		panic(err)
	}

	// ✅ goroutine qui lit le channel
	go func() {
		defer file.Close() // ✅ déplacé ici — sinon file.Close() s'exécute avant les writes
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-logChannel:
				mu.Lock()
				logCache[time.Now().Unix()] = msg
				mu.Unlock()
			}
		}
	}()

	// ✅ ticker dans sa propre goroutine
	go utils.RunTicker(ctx, 2*time.Minute, func() {
		mu.Lock()
		defer mu.Unlock()
		if len(logCache) == 0 {
			return
		}
		for ts, msg := range logCache {
			fmt.Fprintf(file, "[%s] %s\n", time.Unix(ts, 0).Format(time.RFC3339), msg)
		}
		clear(logCache)

	})

	return logChannel // ✅ retourné pour être utilisé depuis n'importe quel package
}
