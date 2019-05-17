package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

const defaultWorkerCount = 3

func main() {
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{
		"/var/log/myproject/myproject.log",
	}
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	defer logger.Sync()
	var stop = make(chan os.Signal)
	signal.Notify(stop, syscall.SIGTERM)
	signal.Notify(stop, syscall.SIGINT)
	workerCount := defaultWorkerCount
	var wg sync.WaitGroup
	wcount := os.Getenv("WORKER_COUNT")
	if count, err := strconv.Atoi(wcount); err == nil {
		workerCount = count
	}
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			logLines(logger, stop)
			wg.Done()
		}()
	}
	wg.Wait()

}

func logLines(logger *zap.Logger, stop chan os.Signal) {
	for {
		select {
		case <-stop:
			return
		default:
			randLevel(logger)(fmt.Sprintf("%d - %s", time.Now().UnixNano(), String(50)),
				zap.String("component", "logger"),
				zap.Int("attempt", 3),
				zap.Duration("backoff", time.Second),
			)
			randLevel(logger)(fmt.Sprintf("%d - %s", time.Now().UnixNano(), String(50)))
		}
	}
}

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset)-1)]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}

func randLevel(logger *zap.Logger) func(msg string, fields ...zap.Field) {
	switch rand.Intn(3) {
	case 0:
		return logger.Info
	case 1:
		return logger.Warn
	case 2:
		return logger.Debug
	default:
		return logger.Error
	}
}
