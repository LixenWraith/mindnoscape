package cli

import (
	"fmt"
	"io"
)

// SystemExit handles the 'system exit' command
func (c *CLI) SystemExit() error {
	c.UI.Println("Exiting...")
	return io.EOF
}

// SystemQuit handles the 'system quit' command
func (c *CLI) SystemQuit() error {
	return c.SystemExit()
}

// SystemUndo handles the 'system undo' command
func (c *CLI) SystemUndo(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: system undo")
	}

	err := c.Data.HistoryManager.Undo()
	if err != nil {
		return err
	}

	c.UI.Success("Undo successful")
	return nil
}

// SystemRedo handles the 'system redo' command
func (c *CLI) SystemRedo(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: system redo")
	}

	err := c.Data.HistoryManager.Redo()
	if err != nil {
		return err
	}

	c.UI.Success("Redo successful")
	return nil
}

// ExecuteSystemCommand routes the system command to the appropriate handler
func (c *CLI) ExecuteSystemCommand(args []string) error {
	if len(args) == 0 {
		return c.SystemInfo(args)
	}

	operation := args[0]
	switch operation {
	case "exit":
		return c.SystemExit()
	case "quit":
		return c.SystemQuit()
	case "undo":
		return c.SystemUndo(args)
	case "redo":
		return c.SystemRedo(args)
	default:
		return fmt.Errorf("unknown system operation: %s", operation)
	}
}
