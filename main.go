package main

import (
	"context"
	"fmt"
	"log"
	"log-ingestion/config"
	"log-ingestion/internal/types"
	"log-ingestion/workers/aggregration"
	"log-ingestion/workers/directorywalker"
	"log-ingestion/workers/ingestion"
	"os/signal"
	"sync"
	"syscall"
)

func run(ctx context.Context) error {
	cfg, err := config.Load("config.json")
	if err != nil {
		return fmt.Errorf("config load error: %w", err)
	}

	// Stage 1: Discovery
	discoveredFiles := make(chan types.LogFile, cfg.DiscoveredFilesChannelSize)

	// Stage 2: Processing
	logCountsChannel := make(chan types.LogCounts, cfg.ProcessedLogCountChannelSize)

	// -----------------------------
	// DIRECTORY WALKERS
	// -----------------------------
	var walkerWg sync.WaitGroup
	sem := make(chan struct{}, cfg.Walker.MaxDiscoveryWorkers)

	for _, folder := range cfg.Walker.LogDirs {
		walkerWg.Add(1)

		go func(dir string) {
			defer walkerWg.Done()

			// acquire slot
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := directorywalker.Walk(ctx, dir, discoveredFiles); err != nil {
				log.Printf("walker error for %s: %v", dir, err)
			}
		}(folder)
	}

	// Close discoveredFiles AFTER walkers finish
	go func() {
		walkerWg.Wait()
		close(discoveredFiles)
	}()

	// -----------------------------
	//  INGESTION WORKERS
	// -----------------------------
	var ingestWg sync.WaitGroup

	for i := 0; i < cfg.Ingestion.MaxIngestionWorkers; i++ {
		ingestWg.Add(1)

		go func() {
			defer ingestWg.Done()

			if err := ingestion.Ingest(ctx, discoveredFiles, logCountsChannel); err != nil {
				log.Printf("ingestion error: %v", err)
			}
		}()
	}

	// Close logCountsChannel AFTER ingestion workers finish
	go func() {
		ingestWg.Wait()
		close(logCountsChannel)
	}()

	// -----------------------------
	// AGGREGATOR
	// -----------------------------
	totalLogCounts := aggregration.Aggregate(ctx,logCountsChannel)
	for service,serviceCount := range(totalLogCounts){
		fmt.Printf("%s Error Counts\n",service)
		for errorLevel,errorCount := range(serviceCount){
			fmt.Printf("%s: %v\n",errorLevel,errorCount)
		}
		fmt.Printf("---------------------\n")
	}

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
