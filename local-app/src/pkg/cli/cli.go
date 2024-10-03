package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"mindnoscape/local-app/src/pkg/adapter"
	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
)

// CLI represents the command-line interface
type CLI struct {
	adapter *adapter.CLIAdapter
	session *model.Session
	stopCh  chan struct{}
	reader  io.Reader
	writer  io.Writer
	logger  *log.Logger
}

// NewCLI creates a new CLI instance
func NewCLI(adapter *adapter.CLIAdapter, logger *log.Logger) (*CLI, error) {
	sessionID, err := adapter.SessionAdd()
	if err != nil {
		return nil, fmt.Errorf("failed to create new session: %v", err)
	}

	cli := &CLI{
		adapter: adapter,
		session: &model.Session{ID: sessionID},
		stopCh:  make(chan struct{}),
		reader:  os.Stdin,
		writer:  os.Stdout,
		logger:  logger,
	}

	logger.Info(context.Background(), "CLI instance created", log.Fields{"sessionID": sessionID})
	return cli, nil
}

// Run starts the CLI and handles user input
func (c *CLI) Run() error {
	fmt.Println("Welcome to Mindnoscape CLI!")
	fmt.Println("Type 'system help' for a list of commands or 'system exit' to quit.")

	for {
		prompt := c.adapter.PromptGet(c.session.ID)
		fmt.Print(prompt)

		input, err := c.readLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			c.logger.Error(context.Background(), "Error reading input", log.Fields{"error": err})
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		// Send raw input to CLIAdapter
		result, err := c.adapter.ProcessInput(c.session.ID, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else if result != nil {
			fmt.Printf("%v\n", result)
		}

		// Check if the command was to exit/quit
		if strings.HasPrefix(strings.ToLower(input), "exit") || strings.HasPrefix(strings.ToLower(input), "quit") {
			break
		}
	}

	c.logger.Info(context.Background(), "CLI stopped", nil)
	return nil
}

// readLine reads a line of input from the reader
func (c *CLI) readLine() (string, error) {
	var line strings.Builder
	for {
		var b [1]byte
		n, err := c.reader.Read(b[:])
		if err != nil {
			if err == io.EOF && line.Len() > 0 {
				return line.String(), nil
			}
			return "", err
		}
		if n == 0 {
			continue
		}
		if b[0] == '\n' {
			return line.String(), nil
		}
		line.WriteByte(b[0])
	}
}

// Stop signals the CLI to stop its main loop
func (c *CLI) Stop() {
	close(c.stopCh)
	c.logger.Info(context.Background(), "CLI stop signal received", nil)

	// Remove the connection when stopping
	c.adapter.SessionDelete(c.session.ID)
}
