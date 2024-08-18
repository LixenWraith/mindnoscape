// Package config provides functionality for loading, saving, and managing
// application configuration settings.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the configuration settings for the application.
type Config struct {
	DatabaseDir         string `json:"database_dir"`
	DatabaseFile        string `json:"database_file"`
	LogFolder           string `json:"log_folder"`
	CommandLog          string `json:"command_log"`
	ErrorLog            string `json:"error_log"`
	DefaultUser         string `json:"default_user"`
	DefaultUserActive   bool   `json:"default_user_active"`
	DefaultUserPassword string `json:"default_user_password"`
}

// Global variables to store the current configuration and its file path.
var (
	currentConfig *Config
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
		defaultConfig := &Config{
			DatabaseDir:       "./data",
			DatabaseFile:      "mindnoscape.db",
			LogFolder:         "./log",
			CommandLog:        "commands.log",
			ErrorLog:          "errors.log",
			DefaultUser:       "default",
			DefaultUserActive: true,
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
	currentConfig = &Config{}
	if err := json.Unmarshal(file, currentConfig); err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	return nil
}

// ConfigSave saves the provided configuration to the JSON file.
func ConfigSave(cfg *Config) error {
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
func ConfigGet() *Config {
	return currentConfig
}
