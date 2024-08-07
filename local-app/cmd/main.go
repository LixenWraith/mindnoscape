package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"mindnoscape/local-app/internal/cli"
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
	var err error
	// Initialize SQLite database
	db, err = sql.Open("sqlite3", "./mindmap.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer cleanup()

	// Initialize storage
	store, err := storage.NewSQLiteStore(db)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize mindmap
	mm, err := mindmap.NewMindMap(store)
	if err != nil {
		log.Fatalf("Failed to create mindmap: %v", err)
	}

	// Initialize readline
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     "/tmp/mindnoscape_history.txt",
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
	}
}
