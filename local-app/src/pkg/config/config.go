// Package config provides functionality for loading, saving, and managing
// application configuration settings.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"mindnoscape/local-app/src/pkg/model"
)

// Global variables to store the current configuration and its file path.
var (
	currentConfig *model.Config
	configPath    = "./data/config.json"
)

// ConfigLoad loads the configuration from the JSON file.
// If the file doesn't exist, it creates a default configuration.
func ConfigLoad() error {
	// Ensure the data directory exists
	dataDir := filepath.Dir(configPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// Check if the config file exists, if not create a default one
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := &model.Config{
			DatabaseDir:         "./data",
			DatabaseFile:        "mindnoscape.db",
			DatabaseType:        "sqlite",
			LogFolder:           "./logs",
			CommandLog:          "commands.log",
			ErrorLog:            "errors.log",
			InfoLog:             "info.log",
			DefaultUser:         "a",
			DefaultUserActive:   true,
			DefaultUserPassword: "",
		}
		if err := ConfigSave(defaultConfig); err != nil {
			return fmt.Errorf("failed to create default config: %v", err)
		}
		currentConfig = defaultConfig
		return nil
	}

	// Read and parse the existing config file
	file, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	// Unmarshal the config from JSON
	currentConfig = &model.Config{}
	if err := json.Unmarshal(file, currentConfig); err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	// Set default database type if not specified
	if currentConfig.DatabaseType == "" {
		currentConfig.DatabaseType = "sqlite"
		if err := ConfigSave(currentConfig); err != nil {
			return fmt.Errorf("failed to save updated config: %v", err)
		}
	}

	return nil
}

// ConfigSave saves the provided configuration to the JSON file.
func ConfigSave(cfg *model.Config) error {
	// Marshal the config to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config: %v", err)
	}

	// Write the JSON data to the config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	return nil
}

// ConfigGet returns the current configuration.
func ConfigGet() *model.Config {
	return currentConfig
}
