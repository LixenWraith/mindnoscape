package mindmap

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"mindnoscape/local-app/internal/models"
	"mindnoscape/local-app/internal/storage"
)

type MindMap struct {
	Root     *models.Node
	Nodes    map[int]*models.Node
	MaxIndex int
	Store    storage.Store
}

func NewMindMap(store storage.Store) (*MindMap, error) {
	fmt.Println("DEBUG: NewMindMap called")

	mm := &MindMap{
		Nodes:    make(map[int]*models.Node),
		Store:    store,
		MaxIndex: 0,
	}

	// Try to load existing root from storage
	root, err := store.GetNode(0)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("DEBUG: Root node not found, creating new one")
			// Create a new root node
			root = models.NewNode(0, "Root")
			root.LogicalIndex = "0"
			if err := store.AddNode(-1, root.Content, root.Extra, root.LogicalIndex); err != nil {
				return nil, fmt.Errorf("failed to create root node: %v", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get root node: %v", err)
		}
	} else {
		fmt.Println("DEBUG: Loaded existing root node")
	}

	mm.Root = root
	mm.Nodes[root.Index] = root

	fmt.Printf("DEBUG: Root node: %+v\n", root)

	// Load all nodes from storage
	if err := mm.loadNodes(); err != nil {
		return nil, fmt.Errorf("failed to load nodes: %v", err)
	}

	mm.assignLogicalIndex(mm.Root, "")

	fmt.Printf("DEBUG: MindMap initialized with %d nodes\n", len(mm.Nodes))

	return mm, nil
}

func (mm *MindMap) loadNodes() error {
	fmt.Println("DEBUG: loadNodes called")

	nodes, err := mm.Store.GetAllNodes()
	if err != nil {
		return err
	}

	fmt.Printf("DEBUG: Retrieved %d nodes from storage\n", len(nodes))

	mm.Nodes = make(map[int]*models.Node)
	var rootNode *models.Node

	// First pass: create all nodes
	for _, node := range nodes {
		mm.Nodes[node.Index] = node
		if node.Index > mm.MaxIndex {
			mm.MaxIndex = node.Index
		}
		node.Children = []*models.Node{} // Initialize Children slice
		fmt.Printf("DEBUG: Loaded node: %+v\n", node)

		// Identify the root node (it should have index 0)
		if node.Index == 0 {
			rootNode = node
		}
	}

	// Second pass: build tree structure
	for _, node := range mm.Nodes {
		if node.Index != 0 { // Skip the root node
			parent := mm.Nodes[node.ParentID]
			if parent != nil {
				parent.Children = append(parent.Children, node)
			} else {
				fmt.Printf("DEBUG: Parent node %d not found for node %d\n", node.ParentID, node.Index)
			}
		}
	}

	// Sort children of each node based on LogicalIndex
	for _, node := range mm.Nodes {
		sort.Slice(node.Children, func(i, j int) bool {
			return node.Children[i].LogicalIndex < node.Children[j].LogicalIndex
		})
	}

	mm.Root = rootNode
	if mm.Root == nil {
		return fmt.Errorf("root node not found")
	}

	fmt.Printf("DEBUG: Tree structure built. Root has %d children\n", len(mm.Root.Children))
	mm.debugPrintStructure(mm.Root, 0)

	return nil
}

func (mm *MindMap) assignLogicalIndex(node *models.Node, prefix string) {
	fmt.Printf("DEBUG: assignLogicalIndex called for node: %s, prefix: %s\n", node.Content, prefix)

	if node == mm.Root {
		node.LogicalIndex = "0"
		prefix = ""
	}

	for i, child := range node.Children {
		child.LogicalIndex = fmt.Sprintf("%s%d", prefix, i+1)
		fmt.Printf("DEBUG: Assigned LogicalIndex %s to node %s\n", child.LogicalIndex, child.Content)
		mm.assignLogicalIndex(child, child.LogicalIndex+".")
	}
}

func (mm *MindMap) findNodeByLogicalIndex(logicalIndex string) *models.Node {
	fmt.Printf("DEBUG: findNodeByLogicalIndex called with logicalIndex: %s\n", logicalIndex)

	if logicalIndex == "0" {
		fmt.Printf("DEBUG: Returning root node\n")
		return mm.Root
	}

	parts := strings.Split(logicalIndex, ".")
	currentNode := mm.Root

	for i, part := range parts {
		index, err := strconv.Atoi(part)
		if err != nil || index < 1 || index > len(currentNode.Children) {
			fmt.Printf("DEBUG: Invalid index at part %d: %s\n", i, part)
			return nil
		}
		currentNode = currentNode.Children[index-1]
		fmt.Printf("DEBUG: Moving to child node: %+v\n", currentNode)
	}

	fmt.Printf("DEBUG: Returning node: %+v\n", currentNode)
	return currentNode
}

func (mm *MindMap) Show(logicalIndex string, showIndex bool) error {
	// Reload nodes from storage to ensure we have the latest data
	if err := mm.loadNodes(); err != nil {
		return fmt.Errorf("failed to reload nodes: %v", err)
	}

	fmt.Println("Mind Map Structure:")
	mm.visualize(mm.Root, "", true, showIndex)

	// Debug output
	fmt.Println("\nDebug: Full mind map structure:")
	mm.debugPrintStructure(mm.Root, 0)

	return nil
}

func (mm *MindMap) visualize(node *models.Node, prefix string, isLast bool, showIndex bool) {
	if node == nil {
		fmt.Println("DEBUG: Attempting to visualize nil node")
		return
	}

	var line strings.Builder

	if node == mm.Root {
		line.WriteString(node.Content)
	} else {
		if isLast {
			line.WriteString(fmt.Sprintf("%s└── ", prefix))
			prefix += "    "
		} else {
			line.WriteString(fmt.Sprintf("%s├── ", prefix))
			prefix += "│   "
		}

		line.WriteString(fmt.Sprintf("%s %s", node.LogicalIndex, node.Content))
		if showIndex {
			line.WriteString(fmt.Sprintf(" [%d]", node.Index))
		}
	}

	// Add extra fields
	var extraFields []string
	for k, v := range node.Extra {
		extraFields = append(extraFields, fmt.Sprintf("%s:%s", k, v))
	}
	if len(extraFields) > 0 {
		sort.Strings(extraFields) // Sort extra fields for consistent output
		line.WriteString(" " + strings.Join(extraFields, " "))
	}

	fmt.Println(line.String())

	for i, child := range node.Children {
		mm.visualize(child, prefix, i == len(node.Children)-1, showIndex)
	}
}

func (mm *MindMap) debugPrintStructure(node *models.Node, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Printf("%s- [%d] %s (LogicalIndex: %s, ParentID: %d)\n", indent, node.Index, node.Content, node.LogicalIndex, node.ParentID)
	for _, child := range node.Children {
		mm.debugPrintStructure(child, depth+1)
	}
}
