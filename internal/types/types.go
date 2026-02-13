package types

import "time"

// LogFile represents metadata for a discovered log file.
type LogFile struct {
	LogFilePath    string
	LogFileSize    int64
	LogFileModTime time.Time
}

type LogCounts map[string]map[string]int