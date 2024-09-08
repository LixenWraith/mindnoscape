// Package main is the entry point for the Mindnoscape application.
// It initializes all components and runs the main program loop.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"mindnoscape/local-app/pkg/adapter"
	"mindnoscape/local-app/pkg/cli"
	"mindnoscape/local-app/pkg/config"
	"mindnoscape/local-app/pkg/data"
	"mindnoscape/local-app/pkg/log"
	"mindnoscape/local-app/pkg/session"
	"mindnoscape/local-app/pkg/storage"
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
		fmt.Println(fmt.Sprintf("Failed to load configuration: %v", err))
		os.Exit(1)
	}
	cfg := config.ConfigGet()

	fmt.Println("DEBUG: Config loaded.")

	// Initialize logger
	logger, err := log.NewLogger(cfg.LogFolder, cfg.CommandLog, cfg.ErrorLog)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			fmt.Println(fmt.Sprintf("Failed to close logger: %v", err))
		}
	}()

	fmt.Println("DEBUG: Logger initialized.")

	// Initialize storage config
	store, err := storage.NewStorage(cfg)
	if err != nil {
		fmt.Printf("Failed to initialize storage: %v", err)
		_ = logger.LogError(fmt.Errorf("failed to initialize storage: %w", err))
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			fmt.Printf("Error closing storage: %v\n", err)
			_ = logger.LogError(fmt.Errorf("failed to close storage: %w", err))
		}
	}()

	fmt.Println("DEBUG: Storage initialized.")

	// Initialize data manager
	dataManager, err := data.NewDataManager(store.UserStore, store.MindmapStore, store.NodeStore, cfg, logger)
	if err != nil {
		_ = logger.LogError(fmt.Errorf("failed to inialize data manager: %w", err))
		os.Exit(1)
	}

	fmt.Println("DEBUG: Data manager initialized.")

	// Initialize session manager
	sessionManager := session.NewSessionManager(dataManager)
	defer sessionManager.StopCleanupRoutine()

	fmt.Println("DEBUG: Session manager initialized.")

	// Initialize adapter manager
	adapterManager := adapter.NewAdapterManager(sessionManager)
	defer adapterManager.Shutdown()

	fmt.Println("DEBUG: Adapter manager initialized.")

	// Initialize cli  adapter
	cliAdapter, err := adapter.NewCLIAdapter(sessionManager)
	if err != nil {
		logger.LogError(fmt.Errorf("failed to initialize CLI adapter: %w", err))
		os.Exit(1)
	}

	fmt.Println("DEBUG: CLI adapter initialized.")

	fmt.Println("Welcome to Mindnoscape! Use 'help' for the list of commands.")

	// Create and run CLI
	cliInstance, err := cli.NewCLI(cliAdapter)
	if err != nil {
		logger.LogError(fmt.Errorf("failed to initiate CLI: %w", err))
		os.Exit(1)
	}

	fmt.Println("DEBUG: CLI instance created")

	// Set up graceful shutdown
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal. Shutting down...")
		cliInstance.Stop()
	}()

	// Run CLI
	if err := cliInstance.Run(); err != nil {
		logger.LogError(fmt.Errorf("CLI error: %w", err))
	}

	fmt.Println("Goodbye!")
}
