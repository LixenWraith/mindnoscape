package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"

	"mindnoscape/local-app/internal/mindmap"
	"mindnoscape/local-app/internal/ui"
)

type CLI struct {
	Mindmap      *mindmap.MindmapManager
	RL           *readline.Instance
	Prompt       string
	History      []string
	HistoryIndex int
	CurrentUser  string
	UI           *ui.UI
}

func NewCLI(mm *mindmap.MindmapManager, rl *readline.Instance) *CLI {
	cli := &CLI{
		Mindmap:      mm,
		RL:           rl,
		Prompt:       "> ",
		History:      []string{},
		HistoryIndex: -1,
		CurrentUser:  "guest",
		UI:           ui.NewUI(os.Stdout, true),
	}
	cli.UpdatePrompt()
	return cli
}

func (c *CLI) SetUser(username string) {
	c.CurrentUser = username
	c.Mindmap.ChangeUser(username)
}

func (c *CLI) UpdatePrompt() {
	mindmapName := ""
	if c.Mindmap.CurrentMindmap != nil {
		mindmapName = c.Mindmap.CurrentMindmap.Root.Content
	}
	prompt := c.UI.GetPromptString(c.CurrentUser, mindmapName)
	c.RL.SetPrompt(prompt)
}

func (c *CLI) ExecuteScript(filename string) error {
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
		command := scanner.Text()
		fmt.Print(c.Prompt + command + "\n")
		args := c.ParseArgs(command)
		err := c.ExecuteCommand(args)
		if err != nil {
			c.UI.PrintCommand(c.UI.GetPromptString(c.CurrentUser, c.Mindmap.CurrentMindmap.Root.Content) + command)
		}
		c.UpdatePrompt()
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading script file: %v", err)
	}

	return nil
}

func (c *CLI) RunInteractive() error {
	c.UpdatePrompt()

	line, err := c.RL.Readline()
	if errors.Is(err, readline.ErrInterrupt) {
		return fmt.Errorf("operation interrupted by user")
	} else if err == io.EOF {
		return fmt.Errorf("end of input reached")
	} else if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}

	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil
	}

	args := c.ParseArgs(line)
	err = c.ExecuteCommand(args)
	if err == nil {
		// Only add successful commands to the history
		c.History = append(c.History, line)
		c.HistoryIndex = len(c.History) - 1

		// Write to history file
		if err := c.appendToHistoryFile(line); err != nil {
			c.UI.Error(fmt.Sprintf("Error: %v", err))
		}
	}

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

func (c *CLI) appendToHistoryFile(line string) error {
	f, err := os.OpenFile(c.RL.Config.HistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := f.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	_, err = f.WriteString(line + "\n")
	return err
}

func (c *CLI) ExecuteCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command provided")
	}

	switch args[0] {
	case "user":
		return c.handleUser(args[1:])
	case "access":
		return c.handleModifyAccess(args[1:])
	case "new":
		return c.handleNewNode(args[1:])
	case "switch":
		err := c.handleChangeMindmap(args[1:])
		if err == nil && c.Mindmap.CurrentMindmap != nil {
			// Update the prompt only if switch was successful and we're in a mindmap
			c.Prompt = fmt.Sprintf("%s > ", c.Mindmap.CurrentMindmap.Root.Content)
			c.RL.SetPrompt(c.Prompt)
		}
		return err
	case "list":
		return c.handleListMindmap(args[1:])
	case "add":
		return c.handleAddNode(args[1:])
	case "del":
		return c.handleDeleteNode(args[1:])
	case "clear":
		return c.handleDeleteMindmap(args[1:])
	case "mod":
		return c.handleModifyNode(args[1:])
	case "move":
		return c.handleMoveNode(args[1:])
	case "insert":
		return c.handleInsertNode(args[1:])
	case "show":
		return c.handleShowMindmap(args[1:])
	case "save":
		return c.handleSaveMindmap(args[1:])
	case "load":
		return c.handleLoadMindmap(args[1:])
	case "sort":
		return c.handleSortNode(args[1:])
	case "find":
		return c.handleFindNode(args[1:])
	case "undo":
		return c.handleUndo(args[1:])
	case "redo":
		return c.handleRedo(args[1:])
	case "help":
		return c.handleHelp(args[1:])
	case "exit", "quit":
		return c.handleExit()
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

// The handle* functions (handleAdd, handleDelete, etc.) will be implemented in commands.go

func (c *CLI) printHelp(command string) {
	if command == "" {
		fmt.Println("Available commands:")
		for cmd := range commandHelp {
			fmt.Printf("  %s\n", cmd)
		}
		fmt.Println("\nUse 'help <command>' for more information about a specific command.")
	} else if help, ok := commandHelp[command]; ok {
		fmt.Println(help)
	} else {
		fmt.Printf("Unknown command: %s\n", command)
	}
}

// commandHelp contains help text for each command.
var commandHelp = map[string]string{
	"user": `Syntax: user [--new/--mod/--del][<username> [password]]
Description: Manages user accounts and authentication.
- If no argument is given: Displays the current username.
- If one argument is given: Prompts for password and switches to the provided user.
- If two arguments are given: Switches to the provided user with the given password.
- Option --new: Creates a new user account.
  - If no additional arguments, prompts for new username and password.
  - If one additional argument, uses it as the username and prompts for password.
  - If two additional arguments, uses them as username and password.
- Option --mod: Modifies an existing user account.
  - If currently logged in, modifies the current user.
  - Otherwise, prompts for the user to modify.
  - Prompts for current password, new username (optional), and new password (optional).
Examples:
  user
  user alice
  user bob password123
  user --new
  user --new charlie
  user --new david password456
  user --mod`,

	"access": `Syntax: access <mindmap name> <public|private>
Description: Sets the access level and visibility of a mindmap to public or private.
- <mindmap name>: The name of the mindmap to modify.
- <public|private>: The new access level and visibility setting for the mindmap.
Example: access "My Project" public`,

	"new": `Syntax: new <mindmap name>
Description: Creates a new mindmap with the specified name as its root node.
Example: new "My Project"`,

	"switch": `Syntax: switch [mindmap name]
Description: Switches to the specified mindmap for subsequent operations. If no mindmap name is provided, switches out of the current mindmap.
Examples: 
  switch "My Project"
  switch`,

	"list": `Syntax: list
Description: Lists all available mindmaps in the database.
- Public mindmaps are marked with '+'.
- Private mindmaps are marked with '-'.
Example: list`,

	"add": `Syntax: add <logical index> <content> [<extra field label>:<extra field value>]... [--index]
Description: Adds a new node as a child of the node at the specified logical index or index.
- <logical index>: The logical index of the parent node.
- <content>: The main content of the new node. Use quotes for content with spaces.
- [<extra field label>:<extra field value>]: Optional extra fields for the node.
- [--index]: Optional flag to use index instead of logical index.
Example: add 1 "New Node" priority:high duration:"2 hours"`,

	"del": `Syntax: del <logical index> [--index]
Description: Deletes the node at the specified logical index or index and all its children.
- <logical index>: The logical index of the node to delete.
- [--index]: Optional flag to use index instead of logical index.
Example: del 1.2`,

	"clear": `Syntax: clear [mindmap name]
Description: Clears the specified mindmap or all mindmaps.
- If no argument is given and currently in a mindmap: Clears the current mindmap and switches out.
- If no argument is given and not in a mindmap: Clears all mindmaps, resulting in a clean database.
- If a mindmap name is given: Clears the specified mindmap. If it's the current mindmap, also switches out.
Examples: 
  clear
  clear "My Project"`,

	"mod": `Syntax: mod <logical index> [content] [<extra field label>:<extra field value>]... [--index]
Description: Modifies the content or extra fields of the node at the specified logical index or index.
- <logical index>: The logical index of the node to modify.
- [content]: Optional new content for the node. Use quotes for content with spaces.
- [<extra field label>:<extra field value>]: Optional extra fields to add or modify.
- [--index]: Optional flag to use index instead of logical index.
Example: mod 1.1 "Updated Content" priority:low duration:`,

	"move": `Syntax: move <source logical index> <target logical index> [--index]
Description: Moves the node at the source logical index to become a child of the node at the target logical index.
- <source logical index>: The logical index of the node to move.
- <target logical index>: The logical index of the new parent node.
- [--index]: Optional flag to use index instead of logical index.
Example: move 1.2 2`,

	"insert": `Syntax: insert <source logical index> <target logical index> [--index]
Description: Inserts the source node and its children before the target node, making them siblings.
- <source logical index>: The logical index of the node to insert.
- <target logical index>: The logical index of the node before which to insert.
- [--index]: Optional flag to use index instead of logical index.
Example: insert 1.1 2`,

	"sort": `Syntax: sort [logical index] [extra field label] [--reverse] [--index]
Description: Sorts the children of the specified node based on content or an extra field.
- [logical index]: Optional logical index of the node whose children to sort. If omitted, sorts all nodes.
- [extra field label]: Optional extra field to sort by. If omitted, sorts by content.
- [--reverse]: Optional flag to sort in descending order.
- [--index]: Optional flag to use index instead of logical index.
Example: sort 1 priority --reverse`,

	"find": `Syntax: find <query> [--index]
Description: Searches for nodes whose content or extra fields contain the specified query.
- <query>: The search term to look for in node content and extra fields.
- [--index]: Optional flag to show node indices in the output.
Example: find "important task" --index`,

	"show": `Syntax: show [logical index] [--index]
Description: Displays the mindmap or a specific subtree.
- [logical index]: Optional logical index of the root node to show. If omitted, shows the entire mindmap.
- [--index]: Optional flag to use index instead of logical index.
Example: show 1.2`,

	"save": `Syntax: save [filename] [json/xml]
Description: Saves the current mindmap to a file in JSON or XML format.
- [filename]: Optional filename to save to. If omitted, saves to "mindmap.json".
- [json/xml]: Optional format to save in. Default is JSON if not specified.
Example: save mymap.xml xml`,

	"load": `Syntax: load [filename] [json/xml]
Description: Loads a mindmap from a JSON or XML file.
- [json/xml]: Optional format to load from. Default is JSON if not specified.
- [filename]: Optional filename to load from. If omitted, loads from "mindmap.json" or "mindmap.xml".
Example: load xml mymap.xml`,

	"quit": `Syntax: quit
Description: Exits the program.`,

	"exit": `Syntax: exit
Description: Exits the program.`,
}
