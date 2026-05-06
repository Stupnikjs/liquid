package logging

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/Stupnikjs/liquid/internal/utils"
)

type AppLogg struct {
	Type    string
	Content string
}

/* define log type instead of string  */

func NewLogger(ctx context.Context, filename string) chan string {
	var mu sync.Mutex

	logChannel := make(chan string, 1000) // ✅ buffered pour éviter les blocages
	logCache := make([]string, 0, 100)

	// check if logs dir exist
	_, err := os.Stat("logs")
	if os.IsNotExist(err) {
		os.Mkdir("logs", 0755)
	}
	pathLog := path.Join("logs", filename)
	file, err := os.OpenFile(pathLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
				logCache = append(logCache, fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), msg))
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
		for _, line := range logCache {
			fmt.Fprintln(file, line)
		}
		file.Sync()
		logCache = logCache[:0]

	})

	return logChannel
}
