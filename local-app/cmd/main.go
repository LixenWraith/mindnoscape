package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	_ "log"
	"os"

	"github.com/chzyer/readline"
	_ "github.com/mattn/go-sqlite3"

	"mindnoscape/local-app/internal/cli"
	"mindnoscape/local-app/internal/config"
	"mindnoscape/local-app/internal/mindmap"
	"mindnoscape/local-app/internal/storage"
	"mindnoscape/local-app/internal/ui"
)

var db *sql.DB

func main() {
	UI := ui.NewUI(os.Stdout, true)
	UI.Println("Welcome to Mindnoscape! Use 'help' for the list of commands.")

	// Load configuration
	if err := config.LoadConfig(); err != nil {
		UI.Error(fmt.Sprintf("Failed to load configuration: %v", err))
		os.Exit(1)
	}
	cfg := config.GetConfig()

	// Clear history
	if err := func() error {
		return os.WriteFile(cfg.HistoryFile, []byte{}, 0644)
	}(); err != nil {
		UI.Error(fmt.Sprintf("Failed to clear history file: %v", err))
	}

	var err error
	// Initialize SQLite database using the path from config
	db, err = sql.Open("sqlite3", cfg.DatabasePath)
	if err != nil {
		UI.Error(fmt.Sprintf("Failed to open database: %v", err))
		os.Exit(1)
	}
	defer cleanup()

	// Initialize storage
	store, err := storage.NewSQLiteStore(db)
	if err != nil {
		UI.Error(fmt.Sprintf("Failed to initialize storage: %v", err))
		os.Exit(1)
	}

	// Initialize mindmap manager with the default user
	mm, err := mindmap.NewMindMapManager(store, "guest")
	if err != nil {
		UI.Error(fmt.Sprintf("Failed to create mindmap manager: %v", err))
		os.Exit(1)
	}

	// Initialize readline with history file from config
	rl, err := readline.NewEx(&readline.Config{
		HistoryFile:     cfg.HistoryFile,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		UI.Error(fmt.Sprintf("Failed to initialize readline: %v", err))
		os.Exit(1)
	}
	defer rl.Close()

	// Initialize CLI
	cli := cli.NewCLI(mm, rl)

	// Check for script arguments
	if len(os.Args) > 1 {
		for _, scriptFile := range os.Args[1:] {
			err := cli.ExecuteScript(scriptFile)
			if err != nil {
				UI.Error(fmt.Sprintf("Error executing script %s: %v", scriptFile, err))
			}
		}
	}

	// Main loop
	for {
		err := cli.Run()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				UI.Println("Use 'exit' or 'quit' to exit the program.")
				continue
			} else if errors.Is(err, io.EOF) {
				break
			} else if err.Error() == "exit requested: EOF" {
				break
			}
			UI.Error(fmt.Sprintf("Error: %v", err))
		}
	}
}

func cleanup() {
	UI := ui.NewUI(os.Stdout, true)
	if db != nil {
		UI.Println("Closing database connection...")
		err := db.Close()
		if err != nil {
			UI.Error(fmt.Sprintf("Error closing database: %v", err))
		}
	}
	UI.Println("Goodbye!")
}
