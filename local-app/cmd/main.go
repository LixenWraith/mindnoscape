// Package main is the entry point for the Mindnoscape application.
// It initializes all components and runs the main program loop.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"mindnoscape/local-app/internal/cli"
	"mindnoscape/local-app/internal/config"
	"mindnoscape/local-app/internal/data"
	"mindnoscape/local-app/internal/log"
	"mindnoscape/local-app/internal/storage"
	"mindnoscape/local-app/internal/ui"
)

// main is the entry point of the application. It initializes all components,
// sets up signal handling, and runs the main program loop.
func main() {
	// Set up channel to receive interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine and channel to handle interrupt signal and signal program exit
	exitChan := make(chan struct{})
	go func() {
		<-sigChan
		close(exitChan)
	}()

	// Initialize UI
	UI := ui.NewUI(os.Stdout, true)
	UI.Message("Welcome to Mindnoscape! Use 'help' for the list of commands.")

	// Load configuration
	if err := config.ConfigLoad(); err != nil {
		UI.Error(fmt.Sprintf("Failed to load configuration: %v", err))
		os.Exit(1)
	}
	cfg := config.ConfigGet()

	// Initialize logger
	logger, err := log.NewLogger(cfg.LogFolder, cfg.CommandLog, cfg.ErrorLog)
	if err != nil {
		UI.Error(fmt.Sprintf("Failed to initialize logger: %v", err))
		os.Exit(1)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			UI.Error(fmt.Sprintf("Failed to close logger: %v", err))
		}
	}()

	// Initialize SQLite database using the path from config
	store, err := storage.NewSQLiteStore(cfg.DatabaseDir, cfg.DatabaseFile)
	if err != nil {
		UI.Error(fmt.Sprintf("Failed to initialize storage: %v", err))
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			UI.Error(fmt.Sprintf("Error closing storage: %v\n", err))
		}
	}()

	// Initialize data manager
	dataManager, err := data.NewManager(store.UserStore, store.MindmapStore, store.NodeStore, cfg)
	if err != nil {
		UI.Error(fmt.Sprintf("Failed to create data manager: %v", err))
		os.Exit(1)
	}

	// Initialize CLI
	c, err := cli.NewCLI(dataManager, logger)
	if err != nil {
		UI.Error(fmt.Sprintf("Failed to initialize CLI: %v", err))
		os.Exit(1)
	}

	// Check for script arguments
	if len(os.Args) > 1 {
		for _, scriptFile := range os.Args[1:] {
			err := c.ExecuteScript(scriptFile)
			if err != nil {
				c.UI.Error(fmt.Sprintf("Error executing script %s: %v", scriptFile, err))
			}
		}
	}

	// Main loop
	for {
		select {
		case <-exitChan:
			return
		default:
			err := c.RunInteractive()
			if err != nil {
				if errors.Is(err, io.EOF) {
					UI.Message("Goodbye!")
					return
				}
				logErr := logger.LogError(fmt.Errorf("main: %s", err.Error()))
				if logErr != nil {
					UI.Error(fmt.Sprintf("Failed to log error: %v", logErr))
				}
				UI.Error(err.Error())
			}
		}
	}
}
