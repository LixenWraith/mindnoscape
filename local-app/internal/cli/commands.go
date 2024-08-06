package cli

import (
	"fmt"
	"strconv"
	"strings"

	"mindnoscape/local-app/internal/storage"
)

func (c *CLI) handleAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: add <parent> <content> [<extra field label>:<extra field value>]...")
	}

	parentIdentifier := args[0]
	content := args[1]
	extra := make(map[string]string)

	for _, arg := range args[2:] {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) == 2 {
			extra[parts[0]] = parts[1]
		}
	}

	useIndex := false
	if _, err := strconv.Atoi(parentIdentifier); err == nil {
		useIndex = true
	}

	err := c.MindMap.AddNode(parentIdentifier, content, extra, useIndex)
	if err != nil {
		return err
	}

	fmt.Println("Node added successfully")
	return nil
}

func (c *CLI) handleDelete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: del <node>")
	}

	identifier := args[0]
	useIndex := false
	if _, err := strconv.Atoi(identifier); err == nil {
		useIndex = true
	}

	err := c.MindMap.DeleteNode(identifier, useIndex)
	if err != nil {
		return err
	}

	fmt.Println("Node deleted successfully")
	return nil
}

func (c *CLI) handleModify(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: mod <node> <content> [<extra field label>:<extra field value>]...")
	}

	identifier := args[0]
	content := args[1]
	extra := make(map[string]string)

	for _, arg := range args[2:] {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) == 2 {
			extra[parts[0]] = parts[1]
		}
	}

	useIndex := false
	if _, err := strconv.Atoi(identifier); err == nil {
		useIndex = true
	}

	err := c.MindMap.ModifyNode(identifier, content, extra, useIndex)
	if err != nil {
		return err
	}

	fmt.Println("Node modified successfully")
	return nil
}

func (c *CLI) handleMove(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: move <source> <target>")
	}

	sourceIdentifier := args[0]
	targetIdentifier := args[1]

	useIndex := false
	if _, err := strconv.Atoi(sourceIdentifier); err == nil {
		if _, err := strconv.Atoi(targetIdentifier); err == nil {
			useIndex = true
		} else {
			return fmt.Errorf("both source and target must be of the same type (index or logical index)")
		}
	}

	err := c.MindMap.MoveNode(sourceIdentifier, targetIdentifier, useIndex)
	if err != nil {
		return err
	}

	fmt.Println("Node moved successfully")
	return nil
}

func (c *CLI) handleShow(args []string) error {
	logicalIndex := ""
	showIndex := false

	for _, arg := range args {
		if arg == "--index" {
			showIndex = true
		} else {
			logicalIndex = arg
		}
	}

	return c.MindMap.Show(logicalIndex, showIndex)
}

func (c *CLI) handleSort(args []string) error {
    identifier := ""
    field := ""
    reverse := false
    useIndex := false
    for i := 0; i < len(args); i++ {
        arg := strings.ToLower(args[i])
        switch arg {
        case "--reverse":
            reverse = true
        case "--index":
            useIndex = true
        default:
            if identifier == "" {
                identifier = args[i]
            } else if field == "" {
                field = args[i]
            }
        }
    }
    err := c.MindMap.Sort(identifier, field, reverse, useIndex)
    if err != nil {
        return err
    }
    fmt.Println("Sorted successfully")
    return nil
}

func (c *CLI) handleSave(args []string) error {
	filename := "mindmap.json"
	format := "json"

	if len(args) >= 1 {
		filename = args[0]
	}
	if len(args) >= 2 {
		format = strings.ToLower(args[1])
	}

	if format != "json" && format != "xml" {
		return fmt.Errorf("unsupported format: %s", format)
	}

	err := storage.SaveToFile(c.MindMap.Store, filename, format)
	if err != nil {
		return err
	}

	fmt.Printf("Mind map saved to %s in %s format\n", filename, format)
	return nil
}

func (c *CLI) handleLoad(args []string) error {
	filename := "mindmap.json"
	format := "json"

	if len(args) >= 1 {
		filename = args[0]
	}
	if len(args) >= 2 {
		format = strings.ToLower(args[1])
	}

	if format != "json" && format != "xml" {
		return fmt.Errorf("unsupported format: %s", format)
	}

	err := storage.LoadFromFile(c.MindMap.Store, filename, format)
	if err != nil {
		return err
	}

	fmt.Printf("Mind map loaded from %s\n", filename)
	return nil
}

func (c *CLI) handleHelp(args []string) error {
	if len(args) > 0 {
		c.printHelp(args[0])
	} else {
		c.printHelp("")
	}
	return nil
}