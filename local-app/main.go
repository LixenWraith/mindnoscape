// Package main implements a command-line mind mapping tool.
package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/chzyer/readline"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Node represents a single node in the mind map.
type Node struct {
	Index        int               `json:"index"`
	Content      string            `json:"content"`
	Children     []*Node           `json:"children,omitempty"`
	Extra        map[string]string `json:"extra,omitempty"`
	LogicalIndex string            `json:"logical_index,omitempty"`
}

// MindMap represents the entire mind map structure.
type MindMap struct {
	Root     *Node
	Nodes    map[int]*Node
	MaxIndex int
}

// NewMindMap creates and initializes a new MindMap structure.
func NewMindMap() *MindMap {
	// Initialize root node
	root := &Node{Index: 1, Content: "Root", Extra: make(map[string]string)}

	return &MindMap{
		Root:     root,
		Nodes:    map[int]*Node{1: root},
		MaxIndex: 1,
	}
}

// getNextIndex increments and returns the next available index for a new node.
func (mm *MindMap) getNextIndex() int {
	mm.MaxIndex++
	return mm.MaxIndex
}

// assignLogicalIndex recursively assigns logical indices to all nodes in the mind map.
func (mm *MindMap) assignLogicalIndex(node *Node, prefix string) {
	// Assign logical index based on parent's prefix
	for i, child := range node.Children {
		if prefix == "" {
			child.LogicalIndex = strconv.Itoa(i + 1)
		} else {
			child.LogicalIndex = fmt.Sprintf("%s.%d", prefix, i+1)
		}
		// Recursively assign to children
		mm.assignLogicalIndex(child, child.LogicalIndex)
	}
}

// AddNode adds a new node to the mind map under the specified parent.
func (mm *MindMap) AddNode(parentIdentifier string, content string, extra map[string]string, useIndex bool) error {
	var parentNode *Node
	if useIndex {
		// Find parent by index
		index, err := strconv.Atoi(parentIdentifier)
		if err != nil {
			return fmt.Errorf("invalid index: %v", err)
		}
		parentNode = mm.Nodes[index]
	} else {
		// Find parent by logical index
		parentNode = mm.findNodeByLogicalIndex(parentIdentifier)
	}

	if parentNode == nil {
		return fmt.Errorf("parent node not found")
	}

	// Create and add new node
	newIndex := mm.getNextIndex()
	newNode := &Node{Index: newIndex, Content: content, Extra: extra}
	parentNode.Children = append(parentNode.Children, newNode)
	mm.Nodes[newIndex] = newNode

	// Update logical indices
	mm.assignLogicalIndex(mm.Root, "")
	return nil
}

// DeleteNode removes a node and all its children from the mind map.
func (mm *MindMap) DeleteNode(identifier string, useIndex bool) error {
	var nodeToDelete *Node
	if useIndex {
		index, err := strconv.Atoi(identifier)
		if err != nil {
			return fmt.Errorf("invalid index: %v", err)
		}
		nodeToDelete = mm.Nodes[index]
	} else {
		nodeToDelete = mm.findNodeByLogicalIndex(identifier)
	}

	if nodeToDelete == nil {
		return fmt.Errorf("node not found")
	}

	if nodeToDelete == mm.Root {
		return fmt.Errorf("cannot delete root node")
	}

	for _, parentNode := range mm.Nodes {
		for i, child := range parentNode.Children {
			if child == nodeToDelete {
				parentNode.Children = append(parentNode.Children[:i], parentNode.Children[i+1:]...)
				mm.deleteNodeRecursive(nodeToDelete)
				mm.assignLogicalIndex(mm.Root, "")
				return nil
			}
		}
	}

	return fmt.Errorf("node not found in parent's children")
}

func (mm *MindMap) deleteNodeRecursive(node *Node) {
	for _, child := range node.Children {
		mm.deleteNodeRecursive(child)
	}
	delete(mm.Nodes, node.Index)
}

// ModifyNode updates the content or extra fields of a node.
func (mm *MindMap) ModifyNode(identifier string, content string, extra map[string]string, useIndex bool) error {
	var node *Node
	if useIndex {
		index, err := strconv.Atoi(identifier)
		if err != nil {
			return fmt.Errorf("invalid index: %v", err)
		}
		node = mm.Nodes[index]
	} else {
		node = mm.findNodeByLogicalIndex(identifier)
	}

	if node == nil {
		return fmt.Errorf("node not found")
	}

	if content != "" {
		node.Content = content
	}
	if node.Extra == nil {
		node.Extra = make(map[string]string)
	}
	for k, v := range extra {
		if v == "" {
			delete(node.Extra, k)
		} else {
			node.Extra[k] = v
		}
	}
	return nil
}

// MoveNode moves a node from its current parent to become a child of another node.
func (mm *MindMap) MoveNode(sourceIdentifier, targetIdentifier string, useIndex bool) error {
	var sourceNode, targetNode *Node
	if useIndex {
		sourceIndex, err := strconv.Atoi(sourceIdentifier)
		if err != nil {
			return fmt.Errorf("invalid source index: %v", err)
		}
		sourceNode = mm.Nodes[sourceIndex]

		targetIndex, err := strconv.Atoi(targetIdentifier)
		if err != nil {
			return fmt.Errorf("invalid target index: %v", err)
		}
		targetNode = mm.Nodes[targetIndex]
	} else {
		sourceNode = mm.findNodeByLogicalIndex(sourceIdentifier)
		targetNode = mm.findNodeByLogicalIndex(targetIdentifier)
	}

	if sourceNode == nil || targetNode == nil {
		return fmt.Errorf("source or target node not found")
	}

	if sourceNode == mm.Root {
		return fmt.Errorf("cannot move root node")
	}

	// Remove the source node from its current parent
	for _, parentNode := range mm.Nodes {
		for i, child := range parentNode.Children {
			if child == sourceNode {
				parentNode.Children = append(parentNode.Children[:i], parentNode.Children[i+1:]...)
				break
			}
		}
	}

	// Add the source node to the target node's children
	targetNode.Children = append(targetNode.Children, sourceNode)
	mm.assignLogicalIndex(mm.Root, "")
	return nil
}

// Insert moves a node to become a sibling of another node.
func (mm *MindMap) Insert(sourceIdentifier, targetIdentifier string, useIndex bool) error {
	var sourceNode, targetNode, targetParent *Node

	if useIndex {
		sourceIndex, err := strconv.Atoi(sourceIdentifier)
		if err != nil {
			return fmt.Errorf("invalid source index: %v", err)
		}
		sourceNode = mm.Nodes[sourceIndex]

		targetIndex, err := strconv.Atoi(targetIdentifier)
		if err != nil {
			return fmt.Errorf("invalid target index: %v", err)
		}
		targetNode = mm.Nodes[targetIndex]
	} else {
		sourceNode = mm.findNodeByLogicalIndex(sourceIdentifier)
		targetNode = mm.findNodeByLogicalIndex(targetIdentifier)
	}

	if sourceNode == nil || targetNode == nil {
		return fmt.Errorf("source or target node not found")
	}

	if sourceNode == mm.Root {
		return fmt.Errorf("cannot move root node")
	}

	// Find target's parent
	for _, node := range mm.Nodes {
		for i, child := range node.Children {
			if child == targetNode {
				targetParent = node
				// Remove source node from its original parent
				for _, parentNode := range mm.Nodes {
					for j, childNode := range parentNode.Children {
						if childNode == sourceNode {
							parentNode.Children = append(parentNode.Children[:j], parentNode.Children[j+1:]...)
							break
						}
					}
				}
				// Insert source node before target node
				targetParent.Children = append(targetParent.Children[:i+1], targetParent.Children[i:]...)
				targetParent.Children[i] = sourceNode
				mm.assignLogicalIndex(mm.Root, "")
				return nil
			}
		}
	}

	return fmt.Errorf("target node's parent not found")
}

// Find searches for nodes containing the given query in their content or extra fields.
func (mm *MindMap) Find(query string) []*Node {
	var results []*Node
	mm.findRecursive(mm.Root, strings.ToLower(query), &results)
	return results
}

func (mm *MindMap) findRecursive(node *Node, query string, results *[]*Node) {
	if strings.Contains(strings.ToLower(node.Content), query) {
		*results = append(*results, node)
	}
	for _, v := range node.Extra {
		if strings.Contains(strings.ToLower(v), query) {
			*results = append(*results, node)
			break
		}
	}
	for _, child := range node.Children {
		mm.findRecursive(child, query, results)
	}
}

// SortChildren sorts the children of a specific node based on a given field.
func (mm *MindMap) SortChildren(identifier string, field string, reverse bool, useIndex bool) error {
	var node *Node
	if useIndex {
		index, err := strconv.Atoi(identifier)
		if err != nil {
			return fmt.Errorf("invalid index: %v", err)
		}
		node = mm.Nodes[index]
	} else {
		node = mm.findNodeByLogicalIndex(identifier)
	}

	if node == nil {
		return fmt.Errorf("node not found")
	}

	mm.sortNodeChildren(node, field, reverse)
	mm.assignLogicalIndex(mm.Root, "")
	return nil
}

func (mm *MindMap) sortNodeChildren(node *Node, field string, reverse bool) {
	sort.Slice(node.Children, func(i, j int) bool {
		var vi, vj string
		if field == "" {
			vi, vj = node.Children[i].Content, node.Children[j].Content
		} else {
			vi = node.Children[i].Extra[field]
			vj = node.Children[j].Extra[field]
		}

		// If the field doesn't exist, fall back to Content
		if vi == "" && vj == "" {
			vi, vj = node.Children[i].Content, node.Children[j].Content
		}

		// Try to compare as numbers if possible
		ni, errI := strconv.ParseFloat(vi, 64)
		nj, errJ := strconv.ParseFloat(vj, 64)
		if errI == nil && errJ == nil {
			if reverse {
				return ni > nj
			}
			return ni < nj
		}

		// Fall back to string comparison
		if reverse {
			return vi > vj
		}
		return vi < vj
	})

	// Move the sorted field to the front of extra fields
	if field != "" {
		for _, child := range node.Children {
			if value, ok := child.Extra[field]; ok {
				// Create a new map with the sorted field as the first entry
				newExtra := make(map[string]string)
				newExtra[field] = value

				// Add the rest of the fields
				for k, v := range child.Extra {
					if k != field {
						newExtra[k] = v
					}
				}

				child.Extra = newExtra
			}
		}
	}

	// Recursively sort children of children
	for _, child := range node.Children {
		mm.sortNodeChildren(child, field, reverse)
	}
}

// Sort sorts the entire mind map or a subtree based on a given field.
func (mm *MindMap) Sort(identifier string, field string, reverse bool, useIndex bool) error {
	var node *Node
	if identifier == "" {
		node = mm.Root
	} else if useIndex {
		index, err := strconv.Atoi(identifier)
		if err != nil {
			return fmt.Errorf("invalid index: %v", err)
		}
		node = mm.Nodes[index]
	} else {
		node = mm.findNodeByLogicalIndex(identifier)
	}

	if node == nil {
		return fmt.Errorf("node not found")
	}

	mm.sortNodeChildren(node, field, reverse)
	mm.assignLogicalIndex(mm.Root, "")
	return nil
}

func (mm *MindMap) findNodeByLogicalIndex(LogicalIndex string) *Node {
	parts := strings.Split(LogicalIndex, ".")
	currentNode := mm.Root

	for _, part := range parts {
		index, err := strconv.Atoi(part)
		if err != nil || index < 1 || index > len(currentNode.Children) {
			return nil
		}
		currentNode = currentNode.Children[index-1]
	}

	return currentNode
}

// Show displays the mind map or a subtree in a hierarchical format.
func (mm *MindMap) Show(LogicalIndex string, showIndex bool) error {
	mm.assignLogicalIndex(mm.Root, "")

	var nodeToShow *Node
	if LogicalIndex == "" {
		nodeToShow = mm.Root
	} else {
		nodeToShow = mm.findNodeByLogicalIndex(LogicalIndex)
		if nodeToShow == nil {
			return fmt.Errorf("node with logical index %s not found", LogicalIndex)
		}
	}

	mm.visualize(nodeToShow, "", true, showIndex)
	return nil
}

func (mm *MindMap) visualize(node *Node, prefix string, isLast bool, showIndex bool) {
	if node == mm.Root {
		for i, child := range node.Children {
			mm.visualize(child, "", i == len(node.Children)-1, showIndex)
		}
		return
	}

	if isLast {
		fmt.Printf("%s└── ", prefix)
		prefix += "    "
	} else {
		fmt.Printf("%s├── ", prefix)
		prefix += "│   "
	}

	fmt.Printf("%s %s", node.LogicalIndex, node.Content)
	if showIndex {
		fmt.Printf(" [%d]", node.Index)
	}

	// Get sorted keys of Extra fields
	var keys []string
	for k := range node.Extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Print Extra fields in alphabetical order
	for _, k := range keys {
		fmt.Printf(" | %s:%s", k, node.Extra[k])
	}
	fmt.Println()

	for i, child := range node.Children {
		mm.visualize(child, prefix, i == len(node.Children)-1, showIndex)
	}
}

func (mm *MindMap) Save(filename string, format string) error {
	if filename == "" {
		filename = "mindmap.json"
	}
	if format == "" {
		format = "json"
	}
	var data []byte
	var err error

	switch strings.ToLower(format) {
	case "json":
		data, err = json.MarshalIndent(mm.Root, "", "  ")
	case "xml":
		data, err = xml.MarshalIndent(mm.Root, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (mm *MindMap) Load(filename string, format string) error {
	if filename == "" {
		filename = "mindmap.json"
	}
	if format == "" {
		format = "json"
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var root Node
	switch strings.ToLower(format) {
	case "json":
		err = json.Unmarshal(data, &root)
	case "xml":
		err = xml.Unmarshal(data, &root)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return err
	}

	mm.Root = &root
	mm.Nodes = make(map[int]*Node)
	mm.indexNodes(mm.Root)
	mm.assignLogicalIndex(mm.Root, "")
	return nil
}

func (mm *MindMap) indexNodes(node *Node) {
	mm.Nodes[node.Index] = node
	for _, child := range node.Children {
		mm.indexNodes(child)
	}
}

// parseArgs splits the input string into arguments, respecting quoted strings.
func parseArgs(input string) []string {
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

/*
func parseExtraFields(args []string) map[string]string {
	extra := make(map[string]string)
	for _, arg := range args {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) == 2 {
			key := strings.Trim(parts[0], "\"")
			value := strings.Trim(parts[1], "\"")
			extra[key] = value
		} else if len(parts) == 1 && strings.HasSuffix(arg, ":") {
			key := strings.TrimSuffix(arg, ":")
			extra[strings.Trim(key, "\"")] = ""
		}
	}
	return extra
}
*/

// printHelp displays help information for commands.
func printHelp(command string) {
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

func main() {
	// Initialize mind map
	mm := NewMindMap()

	// Set up readline for interactive command input
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     "/tmp/mindnoscape_history.txt",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := rl.Close(); err != nil {
			//goland:noinspection GoUnhandledErrorResult
			fmt.Fprintf(os.Stderr, "Error closing readline: %v\n", err)
		}
	}()

	// Main command loop
	for {
		// Read user input
		line, err := rl.Readline()
		if errors.Is(err, readline.ErrInterrupt) {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		// Process and execute command
		line = strings.TrimSpace(line)
		args := parseArgs(line)

		if len(args) == 0 {
			continue
		}

		// Switch on command and call appropriate function
		switch args[0] {
		case "help":
			if len(args) > 1 {
				printHelp(args[1])
			} else {
				printHelp("")
			}

		case "add":
			if len(args) < 3 {
				fmt.Println("Usage: add <logical index> <content> [<extra field label>:<extra field value>]... [--index]")
				continue
			}
			parentIdentifier := args[1]
			content := args[2]
			extra := make(map[string]string)
			useIndex := false
			for i := 3; i < len(args); i++ {
				if args[i] == "--index" {
					useIndex = true
				} else if strings.Contains(args[i], ":") {
					parts := strings.SplitN(args[i], ":", 2)
					extra[parts[0]] = parts[1]
				}
			}
			err := mm.AddNode(parentIdentifier, content, extra, useIndex)
			if err != nil {
				fmt.Println("Error adding node:", err)
			} else {
				fmt.Println("Node added successfully")
			}

		case "mod":
			if len(args) < 2 {
				fmt.Println("Usage: mod <logical index> [content] [<extra field label>:<extra field value>]... [--index]")
				continue
			}
			identifier := args[1]
			content := ""
			extra := make(map[string]string)
			useIndex := false
			for i := 2; i < len(args); i++ {
				if args[i] == "--index" {
					useIndex = true
				} else if strings.Contains(args[i], ":") {
					parts := strings.SplitN(args[i], ":", 2)
					extra[parts[0]] = parts[1]
				} else if content == "" {
					content = args[i]
				}
			}
			err := mm.ModifyNode(identifier, content, extra, useIndex)
			if err != nil {
				fmt.Println("Error modifying node:", err)
			} else {
				fmt.Println("Node modified successfully")
			}

		case "sort":
			identifier := ""
			field := ""
			reverse := false
			useIndex := false
			for i := 1; i < len(args); i++ {
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
			err := mm.Sort(identifier, field, reverse, useIndex)
			if err != nil {
				fmt.Println("Error sorting:", err)
			} else {
				fmt.Println("Sorted successfully")
			}

		case "del":
			if len(args) < 2 {
				fmt.Println("Usage: del <logical index> [--index]")
				continue
			}
			identifier := args[1]
			useIndex := false
			if len(args) > 2 && args[2] == "--index" {
				useIndex = true
			}
			err := mm.DeleteNode(identifier, useIndex)
			if err != nil {
				fmt.Println("Error deleting node:", err)
			} else {
				fmt.Println("Node deleted successfully")
			}

		case "show":
			LogicalIndex := ""
			showIndex := false
			for i := 1; i < len(args); i++ {
				if args[i] == "--index" {
					showIndex = true
					i++ // Skip the next argument as it's the index number
				} else {
					LogicalIndex = args[i]
				}
			}
			err := mm.Show(LogicalIndex, showIndex)
			if err != nil {
				fmt.Println("Error showing mindmap:", err)
			}

		case "save", "load":
			format := "json"
			filename := "mindmap.json"
			for _, arg := range args[1:] {
				lowArg := strings.ToLower(arg)
				if lowArg == "json" || lowArg == "xml" {
					format = lowArg
					if filename == "mindmap.json" {
						filename = "mindmap." + format
					}
				} else {
					filename = arg
				}
			}
			var err error
			if args[0] == "save" {
				err = mm.Save(filename, format)
			} else {
				err = mm.Load(filename, format)
			}
			if err != nil {
				fmt.Printf("Error %sing mindmap: %v\n", args[0], err)
			} else {
				fmt.Printf("Mindmap %sed successfully to/from %s\n", args[0], filename)
			}

		case "move":
			if len(args) < 3 {
				fmt.Println("Usage: move <source logical index> <target logical index> [--index]")
				continue
			}
			sourceIdentifier := args[1]
			targetIdentifier := args[2]
			useIndex := false
			if len(args) > 3 && strings.ToLower(args[3]) == "--index" {
				useIndex = true
			}
			err := mm.MoveNode(sourceIdentifier, targetIdentifier, useIndex)
			if err != nil {
				fmt.Println("Error moving node:", err)
			} else {
				fmt.Println("Node moved successfully")
			}

		case "insert":
			if len(args) < 3 {
				fmt.Println("Usage: insert <source> <target> [--index]")
				continue
			}
			source := args[1]
			target := args[2]
			useIndex := false
			if len(args) > 3 && strings.ToLower(args[3]) == "--index" {
				useIndex = true
			}
			err := mm.Insert(source, target, useIndex)
			if err != nil {
				fmt.Println("Error inserting node:", err)
			} else {
				fmt.Println("Node inserted successfully")
			}

		case "find":
			if len(args) < 2 {
				fmt.Println("Usage: find <query>")
				continue
			}
			query := strings.Join(args[1:], " ")
			results := mm.Find(query)
			if len(results) == 0 {
				fmt.Println("No results found")
			} else {
				fmt.Printf("Found %d results:\n", len(results))
				for _, node := range results {
					fmt.Printf("[%s] %s\n", node.LogicalIndex, node.Content)
					for k, v := range node.Extra {
						fmt.Printf("  %s: %s\n", k, v)
					}
				}
			}

		case "quit", "exit":
			fmt.Println("Goodbye!")
			return

		default:
			fmt.Println("Unknown command. Use 'help' to see available commands.")
		}
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
