package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the root application configuration.
// It is loaded from a JSON file and passed to all services
// that require runtime configuration.
type Config struct {
	Walker                       WalkerConfig    `json:"walker"`
	Ingestion                    IngestionConfig `json:"ingestion"`
	DiscoveredFilesChannelSize   int             `json:"discovered_files_channel_size"`
	ProcessedLogCountChannelSize int             `json:"processed_log_count_channel_size"`
}

// WalkerConfig contains configuration options for the
// directory walker / discovery service responsible for
// finding log files to ingest.
type WalkerConfig struct {
	LogDirs             []string `json:"log_dirs"`
	MaxDiscoveryWorkers int      `json:"max_discovery_workers"`
}

// IngestionConfig contains configuration options for the
// file ingestion service responsible for
// processing discovered log files.
type IngestionConfig struct {
	MaxIngestionWorkers int `json:"max_ingestion_workers"`
}

// Load reads the configuration file from disk, unmarshals
// the JSON into a Config struct, and validates the result.
func Load(path string) (Config, error) {
	// Read the configuration file into memory
	configFile, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("error reading config file: %w", err)
	}

	// Parse the JSON configuration
	var config Config
	if err := json.Unmarshal(configFile, &config); err != nil {
		return Config{}, fmt.Errorf("error parsing config file: %w", err)
	}

	// Validate configuration values before returning
	if err := config.validateConfig(); err != nil {
		return Config{}, err
	}

	return config, nil
}

// validateConfig ensures required configuration values
// are present and valid before the application starts.
func (c *Config) validateConfig() error {
	// At least one log directory must be provided
	if len(c.Walker.LogDirs) == 0 {
		return fmt.Errorf("log directories must not be empty")
	}

	// Discovery workers must be a positive number
	if c.Walker.MaxDiscoveryWorkers <= 0 {
		return fmt.Errorf("max discovery workers must be greater than 0")
	}

	// Discovery workers must be a positive number
	if c.DiscoveredFilesChannelSize <= 0 {
		return fmt.Errorf("discovered files channel size must be greater than 0")
	}

	return nil
}
