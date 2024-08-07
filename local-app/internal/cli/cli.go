package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/chzyer/readline"
	"mindnoscape/local-app/internal/mindmap"
)

type CLI struct {
	MindMap *mindmap.MindMap
	RL      *readline.Instance
}

func NewCLI(mm *mindmap.MindMap, rl *readline.Instance) *CLI {
	return &CLI{
		MindMap: mm,
		RL:      rl,
	}
}

func (c *CLI) Run() error {
	line, err := c.RL.Readline()
	if err == readline.ErrInterrupt {
		return err
	} else if err == io.EOF {
		return err
	} else if err != nil {
		return err
	}

	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil
	}

	args := c.ParseArgs(line)
	return c.ExecuteCommand(args)
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

	switch args[0] {
	case "add":
		return c.handleAdd(args[1:])
	case "del":
		return c.handleDelete(args[1:])
	case "clear":
		return c.handleClear(args[1:])
	case "mod":
		return c.handleModify(args[1:])
	case "move":
		return c.handleMove(args[1:])
	case "show":
		return c.handleShow(args[1:])
	case "save":
		return c.handleSave(args[1:])
	case "load":
		return c.handleLoad(args[1:])
	case "sort":
		return c.handleSort(args[1:])
	case "help":
		return c.handleHelp(args[1:])
	case "exit", "quit":
		fmt.Println("Exiting...")
		err := c.RL.Close()
		if err != nil {
			// Log the error but still proceed with exit
			fmt.Printf("Error closing readline: %v\n", err)
		}
		// Wrap both the potential close error and EOF in a custom error
		return fmt.Errorf("exit requested: %w", io.EOF)
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

	"clear": `Syntax: clear
Description: Clears all nodes from the mind map, leaving only the root node.
Example: clear`,

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

	"sort": `Syntax: sort [logical index] [extra field label] [--reverse] [--index]
Description: Sorts the children of the specified node based on content or an extra field.
- [logical index]: Optional logical index of the node whose children to sort. If omitted, sorts all nodes.
- [extra field label]: Optional extra field to sort by. If omitted, sorts by content.
- [--reverse]: Optional flag to sort in descending order.
- [--index]: Optional flag to use index instead of logical index.
Example: sort 1 priority --reverse`,

	"show": `Syntax: show [logical index] [--index]
Description: Displays the mindmap or a specific subtree.
- [logical index]: Optional logical index of the root node to show. If omitted, shows the entire mindmap.
- [--index]: Optional flag to use index instead of logical index.
Example: show 1.2`,

	"save": `Syntax: save [json/xml] [filename]
Description: Saves the current mindmap to a file in JSON or XML format.
- [json/xml]: Optional format to save in. Default is JSON if not specified.
- [filename]: Optional filename to save to. If omitted, saves to "mindmap.json" or "mindmap.xml".
Example: save xml mymap.xml`,

	"load": `Syntax: load [json/xml] [filename]
Description: Loads a mindmap from a JSON or XML file.
- [json/xml]: Optional format to load from. Default is JSON if not specified.
- [filename]: Optional filename to load from. If omitted, loads from "mindmap.json" or "mindmap.xml".
Example: load xml mymap.xml`,

	"insert": `Syntax: insert <source logical index> <target logical index> [--index]
Description: Inserts the source node and its children before the target node, making them siblings.
- <source logical index>: The logical index of the node to insert.
- <target logical index>: The logical index of the node before which to insert.
- [--index]: Optional flag to use index instead of logical index.
Example: insert 1.1 2`,

	"find": `Syntax: find <query>
Description: Searches for nodes whose content or extra fields contain the specified query.
- <query>: The search term to look for in node content and extra fields.
Example: find important`,

	"quit": `Syntax: quit
Description: Exits the program.`,

	"exit": `Syntax: exit
Description: Exits the program.`,
}
