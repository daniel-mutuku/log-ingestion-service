package ingestion

import (
	"bufio"
	"context"
	"fmt"
	"log-ingestion/internal/types"
	"os"
	"strings"
)

func Ingest(ctx context.Context, logfiles <-chan types.LogFile, logcounts chan<- types.LogCounts) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case lf, ok := <-logfiles:
			if !ok {
				return nil
			}

			// process log file
			lc, err := ProcessLogFile(ctx, lf)
			if err != nil {
				return fmt.Errorf("process file %s error: %w", lf.LogFilePath, err)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case logcounts <- lc:
			}
		}

	}
}

func ProcessLogFile(ctx context.Context, lf types.LogFile) (types.LogCounts, error) {
	file, err := os.Open(lf.LogFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	counts := make(types.LogCounts)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line := scanner.Text()
		
		parts := strings.SplitN(line, " ", 4)
		if len(parts) < 3 {
			continue
		}

		service := parts[1]
		level := parts[2]

		if _, ok := counts[service]; !ok {
			counts[service] = make(map[string]int)
		}

		counts[service][level]++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return counts, nil
}
