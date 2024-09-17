// Package main is the entry point for the Mindnoscape application.
// It initializes all components and runs the main program loop.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"mindnoscape/local-app/src/pkg/adapter"
	"mindnoscape/local-app/src/pkg/cli"
	"mindnoscape/local-app/src/pkg/config"
	"mindnoscape/local-app/src/pkg/data"
	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/session"
	"mindnoscape/local-app/src/pkg/storage"
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

	// Load configuration
	if err := config.ConfigLoad(); err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	cfg := config.ConfigGet()

	// Initialize logger with new info logging
	logger, err := log.NewLogger(cfg, log.LevelInfo)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			logger.Error(context.Background(), "Failed to close logger", log.Fields{"error": err})
		}
	}()

	logger.Info(context.Background(), "Application started", log.Fields{"config": cfg})

	// Initialize storage config
	store, err := storage.NewStorage(cfg, logger)
	if err != nil {
		logger.Error(context.Background(), "Failed to initialize storage", log.Fields{"error": err})
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			logger.Error(context.Background(), "Failed to close storage", log.Fields{"error": err})
		}
	}()

	logger.Info(context.Background(), "Storage initialized", nil)

	// Initialize data manager
	dataManager, err := data.NewDataManager(store, cfg, logger)
	if err != nil {
		logger.Error(context.Background(), "Failed to initialize data manager", log.Fields{"error": err})
		os.Exit(1)
	}

	logger.Info(context.Background(), "Data manager initialized", nil)

	// Initialize session manager
	sessionManager := session.NewSessionManager(dataManager, logger)
	defer sessionManager.StopCleanupRoutine()

	logger.Info(context.Background(), "Session manager initialized", nil)

	// Initialize adapter manager
	adapterManager := adapter.NewAdapterManager(sessionManager, logger)
	defer adapterManager.Shutdown()

	logger.Info(context.Background(), "Adapter manager initialized", nil)

	// Initialize cli adapter
	cliAdapter, err := adapter.NewCLIAdapter(sessionManager, logger)
	if err != nil {
		logger.Error(context.Background(), "Failed to initialize CLI adapter", log.Fields{"error": err})
		os.Exit(1)
	}

	logger.Info(context.Background(), "CLI adapter initialized", nil)

	// Create and run CLI
	cliInstance, err := cli.NewCLI(cliAdapter, logger)
	if err != nil {
		logger.Error(context.Background(), "Failed to initiate CLI", log.Fields{"error": err})
		os.Exit(1)
	}

	logger.Info(context.Background(), "CLI instance created", nil)

	// Set up graceful shutdown
	go func() {
		<-sigChan
		logger.Info(context.Background(), "Received interrupt signal. Shutting down...", nil)
		fmt.Println("\nReceived interrupt signal. Shutting down...")
		cliInstance.Stop()
	}()

	// Run CLI
	if err := cliInstance.Run(); err != nil {
		logger.Error(context.Background(), "CLI error", log.Fields{"error": err})
	}

	logger.Info(context.Background(), "Application shutting down", nil)
	fmt.Println("Goodbye!")
}
