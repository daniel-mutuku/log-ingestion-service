package directorywalker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogFile represents metadata for a discovered log file.
type LogFile struct {
	LogFilePath    string
	LogFileSize    int64
	LogFileModTime time.Time
}

// Walk scans a directory and emits .log files into the provided channel.
// It is designed to run as a goroutine and respects backpressure via
// the buffered channel. It will exit early if the context is cancelled.
func Walk(ctx context.Context, folderPath string, out chan<- LogFile) error {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return fmt.Errorf("read folder %s error: %w", folderPath, err)
	}

	for _, entry := range entries {
		// Check if context has been cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Only process .log files
		if !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Use select to send with context cancellation support
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- LogFile{
			LogFilePath:    filepath.Join(folderPath, entry.Name()),
			LogFileSize:    info.Size(),
			LogFileModTime: info.ModTime(),
		}:
		}
	}

	return nil
}
