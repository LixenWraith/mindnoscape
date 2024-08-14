package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	DatabaseDir  string `json:"database_dir"`
	DatabaseFile string `json:"database_file"`
	LogFolder    string `json:"log_folder"`
	CommandLog   string `json:"command_log"`
	ErrorLog     string `json:"error_log"`
}

var (
	currentConfig *Config
	configPath    = "./data/config.json"
)

func LoadConfig() error {
	// Ensure the data directory exists
	dataDir := filepath.Dir(configPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// Check if the config file exists, if not create a default one
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := &Config{
			DatabaseDir:  "./data",
			DatabaseFile: "mindnoscape.db",
			LogFolder:    "./log",
			CommandLog:   "commands.log",
			ErrorLog:     "errors.log",
		}
		if err := SaveConfig(defaultConfig); err != nil {
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

	currentConfig = &Config{}
	if err := json.Unmarshal(file, currentConfig); err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	return nil
}

func SaveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	return nil
}

func GetConfig() *Config {
	return currentConfig
}
