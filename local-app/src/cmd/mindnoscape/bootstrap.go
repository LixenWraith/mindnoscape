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

// bootstrap initializes and runs the Mindnoscape application.
// It sets up signal handling, loads configuration, initializes components
// (logger, storage, data manager, session manager, adapter manager, CLI adapter),
// runs the CLI, and handles graceful shutdown.
// Returns an error if any part of the initialization or execution fails.
func bootstrap() error {
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
		return fmt.Errorf("failed to load configuration: %v", err)
	}
	cfg := config.ConfigGet()

	// Initialize logger with new info logging
	logger, err := log.NewLogger(cfg, log.LevelInfo)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %v", err)
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
		return fmt.Errorf("failed to initialize storage: %v", err)
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
		return fmt.Errorf("failed to initialize data manager: %v", err)
	}

	logger.Info(context.Background(), "Data manager initialized", nil)

	// Initialize session manager
	sessionManager := session.NewSessionManager(dataManager, logger)
	defer sessionManager.StopCleanupRoutine()

	logger.Info(context.Background(), "Session manager initialized", nil)

	// Initialize adapter manager (which now includes CLI adapter initialization)
	adapterManager, err := adapter.NewAdapterManager(sessionManager, logger)
	if err != nil {
		logger.Error(context.Background(), "Failed to initialize Adapter manager", log.Fields{"error": err})
		return fmt.Errorf("failed to initialize Adapter manager: %v", err)
	}
	defer adapterManager.Shutdown()

	logger.Info(context.Background(), "Adapter manager initialized", nil)

	// Initialize CLI
	cliInstance, err := cli.NewCLI(adapterManager, logger)
	if err != nil {
		logger.Error(context.Background(), "Failed to initialize CLI", log.Fields{"error": err})
		return fmt.Errorf("failed to initialize CLI: %v", err)
	}

	logger.Info(context.Background(), "CLI instance created", nil)

	logger.Info(context.Background(), "CLI instance created", nil)

	// Set up graceful shutdown
	go func() {
		<-sigChan
		logger.Info(context.Background(), "Received interrupt signal. Shutting down...", nil)
		fmt.Println("\nReceived interrupt signal. Shutting down...")
		cliInstance.Stop()
	}()

	// Run cli
	if err := cliInstance.Run(); err != nil {
		logger.Error(context.Background(), "CLI error", log.Fields{"error": err})
		return fmt.Errorf("CLI error: %v", err)
	}

	logger.Info(context.Background(), "Application shutting down", nil)
	fmt.Println("Goodbye!")

	return nil
}
