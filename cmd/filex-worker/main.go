package main

import (
	"context"
	"filex.local/filex/internal/bootstrap"
	"filex.local/filex/internal/worker"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func run() error {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println("filex-worker 0.1.0-dev")
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	c, p, e := bootstrap.Open(ctx)
	if e != nil {
		return e
	}
	defer p.Close()
	w, e := worker.New(p, c.MasterKEK)
	if e != nil {
		return e
	}
	if len(os.Args) == 2 && os.Args[1] == "--once" {
		return w.Step(ctx)
	}
	var wg sync.WaitGroup
	for name, task := range w.Tasks() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				step, cancel := context.WithTimeout(ctx, 5*time.Minute)
				err := task(step)
				cancel()
				if err != nil && ctx.Err() == nil {
					slog.Error("worker task failed; durable work remains retryable", "task", name)
				}
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	}
	wg.Wait()
	return nil

}
func main() {
	if e := run(); e != nil {
		slog.Error("worker stopped")
		os.Exit(1)
	}
}
