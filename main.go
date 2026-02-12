package main

import (
	"context"
	"fmt"
	"log"
	"log-ingestion/config"
	"log-ingestion/workers/directorywalker"
	"os/signal"
	"sync"
	"syscall"
)

func run(ctx context.Context) error {
	cfg, err := config.Load("config.json")
	if err != nil {
		return fmt.Errorf("config load error: %w", err)
	}

	discoveredFiles := make(chan directorywalker.LogFile, cfg.DiscoveredFilesChannelSize)

	// Semaphore to limit the number of concurrent directory walkers
	sem := make(chan struct{}, cfg.Walker.MaxDiscoveryWorkers)

	var wg sync.WaitGroup

	for _, folder := range cfg.Walker.LogDirs {
		wg.Add(1)

		go func(dir string) {
			defer wg.Done()

			// Acquire a slot in the worker pool
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := directorywalker.Walk(ctx, dir, discoveredFiles); err != nil {
				log.Printf("walker error for %s: %v", dir, err)
			}
		}(folder)
	}

	// Close the channel after all walkers complete or context is cancelled
	go func() {
		defer close(discoveredFiles)
		wg.Wait()
	}()

	// TODO: start ingestion workers that consume discoveredFiles

	return nil
}

func main() {
	// Create a root context with signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatalf("application failed: %v", err)
	}
}
