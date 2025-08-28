package process

import (
	"context"
	"log/slog"
	"sync"
)

var (
	ctx, cancel = context.WithCancel(context.Background())
	wg sync.WaitGroup
)

func Shutdown() {
	slog.Info("Shutting down")
	cancel()
}

func Done() <-chan struct{} {
	return ctx.Done()
}

func Wait() {
	slog.Info("Waiting for goroutines to stop")
	wg.Wait()
}

func Go(fn func()) {
	wg.Add(1)
	go func() {
		fn()
		wg.Done()
	}()
}
