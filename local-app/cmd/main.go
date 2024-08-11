package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"mindnoscape/local-app/internal/cli"
	"mindnoscape/local-app/internal/config"
	"mindnoscape/local-app/internal/mindmap"
	"mindnoscape/local-app/internal/storage"

	"github.com/chzyer/readline"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func cleanup() {
	if db != nil {
		fmt.Println("Closing database connection...")
		err := db.Close()
		if err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
	fmt.Println("Goodbye!")
}

func main() {
	fmt.Println("Welcome to Mindnoscape! Use 'help' for the list of commands.")

	// Load configuration
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	cfg := config.GetConfig()

	// Clear history
	if err := func() error {
		return os.WriteFile(cfg.HistoryFile, []byte{}, 0644)
	}(); err != nil {
		log.Printf("Failed to clear history file: %v", err)
	}

	var err error
	// Initialize SQLite database using the path from config
	db, err = sql.Open("sqlite3", cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer cleanup()

	// Initialize storage
	store, err := storage.NewSQLiteStore(db)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize mindmap manager
	mm, err := mindmap.NewMindMapManager(store)
	if err != nil {
		log.Fatalf("Failed to create mindmap manager: %v", err)
	}

	// Initialize readline with history file from config
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     cfg.HistoryFile,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		log.Fatalf("Failed to initialize readline: %v", err)
	}
	defer rl.Close()

	// Initialize CLI
	cli := cli.NewCLI(mm, rl)

	// Main loop
	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			fmt.Println("Use 'exit' or 'quit' to exit the program.")
			continue
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		args := cli.ParseArgs(line)
		if err := cli.ExecuteCommand(args); err != nil {
			if errors.Is(err, io.EOF) {
				break // Exit the loop if ExecuteCommand returns an error containing io.EOF
			}
			fmt.Println("Error:", err)
		}

		// Update the prompt after each command
		rl.SetPrompt(cli.Prompt)
	}
}
