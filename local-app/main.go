package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Node struct {
	Index    int               `json:"index"`
	Content  string            `json:"content"`
	Children []*Node           `json:"children,omitempty"`
	Extra    map[string]string `json:"extra,omitempty"`
}

type MindMap struct {
	Root  *Node
	Nodes map[int]*Node
	MaxIndex   int
}

func NewMindMap() *MindMap {
	root := &Node{Index: 1, Content: "Root", Extra: make(map[string]string)}
	return &MindMap{
		Root:     root,
		Nodes:    map[int]*Node{1: root},
		MaxIndex: 1,
	}
}

func (mm *MindMap) getNextIndex() int {
	mm.MaxIndex++
	return mm.MaxIndex
}

func (mm *MindMap) AddNode(parentIndex int, content string, extra map[string]string) error {
	parent, ok := mm.Nodes[parentIndex]
	if !ok {
		return fmt.Errorf("parent node with index %d not found", parentIndex)
	}

	newIndex := mm.getNextIndex()
	newNode := &Node{Index: newIndex, Content: content, Extra: extra}
	parent.Children = append(parent.Children, newNode)
	mm.Nodes[newIndex] = newNode
	return nil
}


func (mm *MindMap) DeleteNode(index int) error {
	if index == 1 {
		return fmt.Errorf("cannot delete root node")
	}

	node, ok := mm.Nodes[index]
	if !ok {
		return fmt.Errorf("node with index %d not found", index)
	}

	for _, parentNode := range mm.Nodes {
		for i, child := range parentNode.Children {
			if child.Index == index {
				parentNode.Children = append(parentNode.Children[:i], parentNode.Children[i+1:]...)
				break
			}
		}
	}

	mm.deleteNodeRecursive(node)

	return nil
}

func (mm *MindMap) deleteNodeRecursive(node *Node) {
	for _, child := range node.Children {
		mm.deleteNodeRecursive(child)
	}
	delete(mm.Nodes, node.Index)
}

func (mm *MindMap) ModifyNode(index int, content string, extra map[string]string) error {
	node, ok := mm.Nodes[index]
	if !ok {
		return fmt.Errorf("node with index %d not found", index)
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

func (mm *MindMap) MoveNode(sourceIndex, targetIndex int) error {
	sourceNode, ok := mm.Nodes[sourceIndex]
	if !ok {
		return fmt.Errorf("source node with index %d not found", sourceIndex)
	}

	targetNode, ok := mm.Nodes[targetIndex]
	if !ok {
		return fmt.Errorf("target node with index %d not found", targetIndex)
	}

	if sourceIndex == 1 {
		return fmt.Errorf("cannot move root node")
	}

	// Remove the source node from its current parent
	for _, parentNode := range mm.Nodes {
		for i, child := range parentNode.Children {
			if child.Index == sourceIndex {
				parentNode.Children = append(parentNode.Children[:i], parentNode.Children[i+1:]...)
				break
			}
		}
	}

	// Add the source node to the target node's children
	targetNode.Children = append(targetNode.Children, sourceNode)

	return nil
}

func (mm *MindMap) SortChildren(node *Node, field string, reverse bool) {
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

	// Recursively sort children
	for _, child := range node.Children {
		mm.SortChildren(child, field, reverse)
	}
}

func (mm *MindMap) Sort(index int, field string, reverse bool) error {
	if index == 0 {
		mm.SortChildren(mm.Root, field, reverse)
		return nil
	}

	node, ok := mm.Nodes[index]
	if !ok {
		return fmt.Errorf("node with index %d not found", index)
	}

	mm.SortChildren(node, field, reverse)
	return nil
}

func (mm *MindMap) Show(index int) {
	var node *Node
	if index == 0 {
		node = mm.Root
	} else {
		var ok bool
		node, ok = mm.Nodes[index]
		if !ok {
			fmt.Printf("Node with index %d not found\n", index)
			return
		}
	}
	mm.visualize(node, "", true)
}

func (mm *MindMap) visualize(node *Node, prefix string, isLast bool) {
	if isLast {
		fmt.Printf("%s└── [%d] %s", prefix, node.Index, node.Content)
	} else {
		fmt.Printf("%s├── [%d] %s", prefix, node.Index, node.Content)
	}

	for k, v := range node.Extra {
		fmt.Printf(" | %s:%s", k, v)
	}
	fmt.Println()

	if isLast {
		prefix += "    "
	} else {
		prefix += "│   "
	}

	for i, child := range node.Children {
		mm.visualize(child, prefix, i == len(node.Children)-1)
	}
}

func (mm *MindMap) Save(filename string) error {
	data, err := json.MarshalIndent(mm.Root, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

func (mm *MindMap) Load(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var root Node
	err = json.Unmarshal(data, &root)
	if err != nil {
		return err
	}

	mm.Root = &root
	mm.Nodes = make(map[int]*Node)
	mm.indexNodes(mm.Root)
	return nil
}

func (mm *MindMap) indexNodes(node *Node) {
	mm.Nodes[node.Index] = node
	for _, child := range node.Children {
		mm.indexNodes(child)
	}
}

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

func main() {
	mm := NewMindMap()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		scanner.Scan()
		input := scanner.Text()
		args := parseArgs(input)

		if len(args) == 0 {
			continue
		}

		switch args[0] {
		case "add":
			if len(args) < 3 {
				fmt.Println("Usage: add <index> <content> [<extra field label>:<extra field value>]...")
				continue
			}
			index, err := strconv.Atoi(args[1])
			if err != nil {
				fmt.Println("Invalid index:", err)
				continue
			}
			content := args[2]
			extra := parseExtraFields(args[3:])
			err = mm.AddNode(index, content, extra)
			if err != nil {
				fmt.Println("Error adding node:", err)
			} else {
				fmt.Println("Node added successfully")
			}

		case "mod":
			if len(args) < 2 {
				fmt.Println("Usage: mod <index> [content] [<extra field label>:<extra field value>]...")
				continue
			}
			index, err := strconv.Atoi(args[1])
			if err != nil {
				fmt.Println("Invalid index:", err)
				continue
			}
			content := ""
			extraStart := 2
			if len(args) > 2 && !strings.Contains(args[2], ":") {
				content = args[2]
				extraStart = 3
			}
			extra := parseExtraFields(args[extraStart:])
			err = mm.ModifyNode(index, content, extra)
			if err != nil {
				fmt.Println("Error modifying node:", err)
			} else {
				fmt.Println("Node modified successfully")
			}

		case "move":
			if len(args) != 3 {
				fmt.Println("Usage: move <source index> <target index>")
				continue
			}
			sourceIndex, err := strconv.Atoi(args[1])
			if err != nil {
				fmt.Println("Invalid source index:", err)
				continue
			}
			targetIndex, err := strconv.Atoi(args[2])
			if err != nil {
				fmt.Println("Invalid target index:", err)
				continue
			}
			err = mm.MoveNode(sourceIndex, targetIndex)
			if err != nil {
				fmt.Println("Error moving node:", err)
			} else {
				fmt.Println("Node moved successfully")
			}

		case "del":
			if len(args) != 2 {
				fmt.Println("Usage: del <index>")
				continue
			}
			index, err := strconv.Atoi(args[1])
			if err != nil {
				fmt.Println("Invalid index:", err)
				continue
			}
			err = mm.DeleteNode(index)
			if err != nil {
				fmt.Println("Error deleting node:", err)
			} else {
				fmt.Println("Node deleted successfully")
			}

		case "sort":
			index := 0
			field := ""
			reverse := false
			for i := 1; i < len(args); i++ {
				if args[i] == "--reverse" {
					reverse = true
				} else if index == 0 {
					if parsedIndex, err := strconv.Atoi(args[i]); err == nil {
						index = parsedIndex
					} else {
						field = args[i]
					}
				} else {
					field = args[i]
				}
			}
			err := mm.Sort(index, field, reverse)
			if err != nil {
				fmt.Println("Error sorting:", err)
			} else {
				fmt.Println("Sorted successfully")
			}

		case "show":
			index := 0
			if len(args) > 1 {
				var err error
				index, err = strconv.Atoi(args[1])
				if err != nil {
					fmt.Println("Invalid index:", err)
					continue
				}
			}
			mm.Show(index)

		case "save":
			filename := "mindmap.json"
			if len(args) > 1 {
				filename = args[1]
			}
			err := mm.Save(filename)
			if err != nil {
				fmt.Println("Error saving mindmap:", err)
			} else {
				fmt.Println("Mindmap saved successfully")
			}

		case "load":
			filename := "mindmap.json"
			if len(args) > 1 {
				filename = args[1]
			}
			err := mm.Load(filename)
			if err != nil {
				fmt.Println("Error loading mindmap:", err)
			} else {
				fmt.Println("Mindmap loaded successfully")
			}

		case "quit", "exit":
			fmt.Println("Goodbye!")
			return

		default:
			fmt.Println("Unknown command. Available commands: add, del, mod, move, show, save, load, quit, exit")
		}
	}
}
