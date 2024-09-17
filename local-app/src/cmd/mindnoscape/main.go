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
	logger, err := log.NewLogger(cfg, true) // Enable info logging
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			fmt.Printf("Failed to close logger: %v\n", err)
		}
	}()

	logger.LogInfo(context.Background(), fmt.Sprintf("Application started", map[string]interface{}{"config": cfg}))

	// Initialize storage config
	store, err := storage.NewStorage(cfg)
	if err != nil {
		logger.LogError(context.Background(), fmt.Errorf("failed to initialize storage: %w", err))
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			logger.LogError(context.Background(), fmt.Errorf("failed to close storage: %w", err))
		}
	}()

	logger.LogInfo(context.Background(), "Storage initialized")

	// Initialize data manager
	dataManager, err := data.NewDataManager(store, cfg, logger)
	if err != nil {
		logger.LogError(context.Background(), fmt.Errorf("failed to initialize data manager: %w", err))
		os.Exit(1)
	}

	logger.LogInfo(context.Background(), "Data manager initialized")

	// Initialize session manager
	sessionManager := session.NewSessionManager(dataManager, logger)
	defer sessionManager.StopCleanupRoutine()

	logger.LogInfo(context.Background(), "Session manager initialized")

	// Initialize adapter manager
	adapterManager := adapter.NewAdapterManager(sessionManager, logger)
	defer adapterManager.Shutdown()

	logger.LogInfo(context.Background(), "Adapter manager initialized")

	// Initialize cli adapter
	cliAdapter, err := adapter.NewCLIAdapter(sessionManager, logger)
	if err != nil {
		logger.LogError(context.Background(), fmt.Errorf("failed to initialize CLI adapter: %w", err))
		os.Exit(1)
	}

	logger.LogInfo(context.Background(), "CLI adapter initialized")

	// Create and run CLI
	cliInstance, err := cli.NewCLI(cliAdapter, logger)
	if err != nil {
		logger.LogError(context.Background(), fmt.Errorf("failed to initiate CLI: %w", err))
		os.Exit(1)
	}

	logger.LogInfo(context.Background(), "CLI instance created")

	// Set up graceful shutdown
	go func() {
		<-sigChan
		logger.LogInfo(context.Background(), "Received interrupt signal. Shutting down...")
		fmt.Println("\nReceived interrupt signal. Shutting down...")
		cliInstance.Stop()
	}()

	// Run CLI
	if err := cliInstance.Run(); err != nil {
		logger.LogError(context.Background(), fmt.Errorf("CLI error: %w", err))
	}

	logger.LogInfo(context.Background(), "Application shutting down")
	fmt.Println("Goodbye!")
}
