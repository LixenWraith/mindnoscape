package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"mindnoscape/local-app/internal/data"
	"mindnoscape/local-app/internal/log"
	"mindnoscape/local-app/internal/ui"
)

type CLI struct {
	Data        *data.Manager
	Prompt      string
	CurrentUser string
	UI          *ui.UI
	Logger      *log.Logger
}

func NewCLI(dataManager *data.Manager, logger *log.Logger) (*CLI, error) {
	cli := &CLI{
		Data:        dataManager,
		CurrentUser: "",
		UI:          ui.NewUI(os.Stdout, true),
		Logger:      logger,
	}
	cli.UpdatePrompt()
	return cli, nil
}

func (c *CLI) UpdatePrompt() {
	mindmapName := ""
	if c.Data.MindmapManager.CurrentMindmap != nil {
		mindmapName = c.Data.MindmapManager.CurrentMindmap.Name
	}
	c.CurrentUser = c.Data.UserManager.UserGet()
	c.Prompt = c.UI.GetPromptString(c.CurrentUser, mindmapName)
}

func (c *CLI) PrintPrompt() {
	c.UI.Print(c.Prompt)
}

func (c *CLI) promptForInput(prompt string) (string, error) {
	input, err := c.UI.ReadLine(prompt)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func (c *CLI) promptForPassword(prompt string) (string, error) {
	return c.UI.ReadPassword(prompt)
}

func (c *CLI) ExecuteScript(filename string) error {
	c.UpdatePrompt()

	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open script file: %v", err)
	}
	defer func() {
		closeErr := f.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		c.PrintPrompt()
		command := scanner.Text()
		c.UI.Println(command)

		strippedPrompt := stripColorCodes(c.Prompt)
		logEntry := fmt.Sprintf("%s [%s] %s", strippedPrompt, filename, command)
		if err := c.Logger.LogCommand(logEntry); err != nil {
			c.UI.Warning(fmt.Sprintf("Failed to log command: %v", err))
		}

		args := c.ParseArgs(command)
		if err := c.ExecuteCommand(args); err != nil {
			strippedErr := stripColorCodes(err.Error())
			if logErr := c.Logger.LogError(fmt.Errorf("%s", strippedErr)); logErr != nil {
				c.UI.Warning(fmt.Sprintf("Failed to log error: %v", logErr))
			}
			c.UI.Error(err.Error())
		}
		c.UpdatePrompt()
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading script file: %v", err)
	}

	return nil

}

func (c *CLI) RunInteractive() error {
	line, err := c.UI.ReadLine(c.Prompt)
	if err != nil {
		if err == io.EOF || err == ui.ErrInterrupted {
			return io.EOF
		}
		return fmt.Errorf("error reading input: %v", err)
	}

	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil
	}

	// Strip color codes from prompt for logging
	strippedPrompt := stripColorCodes(c.Prompt)
	logEntry := fmt.Sprintf("%s%s", strippedPrompt, line)

	err = c.Logger.LogCommand(logEntry)
	if err != nil {
		c.UI.Warning(fmt.Sprintf("Failed to log command: %v", err))
	}

	args := c.ParseArgs(line)
	err = c.ExecuteCommand(args)
	if err != nil {
		// Strip color codes from error message before logging
		strippedErr := stripColorCodes(err.Error())
		if logErr := c.Logger.LogError(fmt.Errorf("%s", strippedErr)); logErr != nil {
			c.UI.Warning(fmt.Sprintf("Failed to log error: %v", logErr))
		}
		if err == io.EOF {
			return err
		}
	}

	c.UpdatePrompt()

	return err
}

func (c *CLI) ParseArgs(input string) []string {
	var args []string
	var currentArg strings.Builder
	inQuotes := false

	for _, char := range input {
		switch char {
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if !inQuotes {
				if currentArg.Len() > 0 {
					args = append(args, currentArg.String())
					currentArg.Reset()
				}
			} else {
				currentArg.WriteRune(char)
			}
		default:
			currentArg.WriteRune(char)
		}
	}

	if currentArg.Len() > 0 {
		args = append(args, currentArg.String())
	}

	return args
}

func (c *CLI) ExecuteCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command provided")
	}

	// Expand concise command to full command
	if len(args) >= 2 {
		args[0], args[1] = c.expandCommand(args[0], args[1])
	}

	var err error
	switch args[0] {
	case "user":

		err = c.ExecuteUserCommand(args[1:])
	case "mindmap":
		err = c.ExecuteMindmapCommand(args[1:])
	case "node":

		err = c.ExecuteNodeCommand(args[1:])
	case "system":

		err = c.ExecuteSystemCommand(args[1:])
	case "help":

		err = c.HandleHelp(args[1:])
	default:

		err = fmt.Errorf("unknown command: %s", args[0])
	}

	return err
}

// expandCommand converts concise (one letter) commands and operations to the long (complete string) format
func (c *CLI) expandCommand(scope, operation string) (string, string) {
	expandedScope := scope
	expandedOperation := operation

	// Expand scope if it's a single letter
	if len(scope) == 1 {
		switch scope {
		case "s":
			expandedScope = "system"
		case "u":
			expandedScope = "user"
		case "m":
			expandedScope = "mindmap"
		case "n":
			expandedScope = "node"
		}
	}

	// Expand operation if it's a single letter
	if len(operation) == 1 {
		switch expandedScope {
		case "user":
			switch operation {
			case "a":
				expandedOperation = "add"
			case "u":
				expandedOperation = "update"
			case "d":
				expandedOperation = "delete"
			case "s":
				expandedOperation = "select"
			case "l":
				expandedOperation = "list"
			}
		case "mindmap":
			switch operation {
			case "a":
				expandedOperation = "add"
			case "u":
				expandedOperation = "update"
			case "d":
				expandedOperation = "delete"
			case "p":
				expandedOperation = "permission"
			case "i":
				expandedOperation = "import"
			case "e":
				expandedOperation = "export"
			case "s":
				expandedOperation = "select"
			case "l":
				expandedOperation = "list"
			case "v":
				expandedOperation = "view"
			case "c":
				expandedOperation = "connect"
			}
		case "node":
			switch operation {
			case "a":
				expandedOperation = "add"
			case "u":
				expandedOperation = "update"
			case "m":
				expandedOperation = "move"
			case "d":
				expandedOperation = "delete"
			case "f":
				expandedOperation = "find"
			case "s":
				expandedOperation = "sort"
			case "c":
				expandedOperation = "connect"
			case "-":
				expandedOperation = "undo"
			case "+":
				expandedOperation = "redo"
			}
		case "system":
			switch operation {
			case "e":
				expandedOperation = "exit"
			case "q":
				expandedOperation = "quit"
			}
		}
	}

	return expandedScope, expandedOperation
}

// stripColorCodes removes ANSI color codes and UI tags from the input string
func stripColorCodes(input string) string {
	// Remove ANSI color codes
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	withoutAnsi := ansiRegex.ReplaceAllString(input, "")

	// Remove UI tags (assuming they're in the format {{tag}})
	uiTagRegex := regexp.MustCompile(`\{\{[^}]+\}\}`)
	return uiTagRegex.ReplaceAllString(withoutAnsi, "")
}
