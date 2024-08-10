package mindmap

import (
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

type MindMapManager struct {
	Store          storage.Store
	MindMaps       map[string]*MindMap
	CurrentMindMap *MindMap
}

func NewMindMapManager(store storage.Store) (*MindMapManager, error) {
	mm := &MindMapManager{
		Store:    store,
		MindMaps: make(map[string]*MindMap),
	}

	// Load existing mindmaps from storage
	mindmaps, err := store.GetAllMindMaps()
	if err != nil {
		return nil, fmt.Errorf("failed to load mindmaps: %v", err)
	}

	for _, mindmapName := range mindmaps {
		mm.MindMaps[mindmapName] = &MindMap{
			Nodes: make(map[int]*models.Node),
		}
	}

	fmt.Printf("Loaded %d mindmaps\n", len(mm.MindMaps))

	return mm, nil
}

func (mm *MindMapManager) CreateNewMindMap(name string) error {
	if _, exists := mm.MindMaps[name]; exists {
		return fmt.Errorf("mindmap '%s' already exists", name)
	}

	newMindMap := &MindMap{
		Nodes: make(map[int]*models.Node),
	}

	// Create root node
	root := models.NewNode(0, name)
	root.ParentID = -1
	root.LogicalIndex = "0"
	newMindMap.Root = root
	newMindMap.Nodes[0] = root

	// Add to storage
	if err := mm.Store.AddMindMap(name); err != nil {
		return fmt.Errorf("failed to add mindmap to storage: %v", err)
	}

	if err := mm.Store.AddNode(name, -1, root.Content, root.Extra, root.LogicalIndex); err != nil {
		return fmt.Errorf("failed to add root node: %v", err)
	}

	mm.MindMaps[name] = newMindMap
	mm.CurrentMindMap = newMindMap

	fmt.Printf("Created new mindmap '%s'\n", name)
	return nil
}

func (mm *MindMapManager) SwitchMindMap(name string) error {
	mindmap, exists := mm.MindMaps[name]
	if !exists {
		return fmt.Errorf("mindmap '%s' does not exist", name)
	}

	mm.CurrentMindMap = mindmap

	// Debug: Print raw DB structure
	if err := mm.Store.DebugPrintDBStructure(name); err != nil {
		fmt.Printf("Error printing DB structure: %v\n", err)
	}

	// Load nodes for the switched mindmap
	if err := mm.loadNodesForMindMap(name); err != nil {
		return fmt.Errorf("failed to load nodes for mindmap '%s': %v", name, err)
	}

	return nil
}

func (mm *MindMapManager) loadNodesForMindMap(name string) error {
	nodes, err := mm.Store.GetAllNodesForMindMap(name)
	if err != nil {
		return err
	}

	mindmap := mm.MindMaps[name]
	mindmap.Nodes = make(map[int]*models.Node)
	mindmap.MaxIndex = 0

	for _, node := range nodes {
		mindmap.Nodes[node.Index] = node
		if node.Index > mindmap.MaxIndex {
			mindmap.MaxIndex = node.Index
		}
		fmt.Printf("Loaded node: Index=%d, ParentID=%d, Content='%s', LogicalIndex='%s'\n",
			node.Index, node.ParentID, node.Content, node.LogicalIndex)

		if node.Index == 0 {
			mindmap.Root = node
			fmt.Printf("Root node identified: Index=%d, Content='%s'\n", node.Index, node.Content)
		}
	}

	// Build tree structure
	for _, node := range mindmap.Nodes {
		if node != mindmap.Root {
			parent, exists := mindmap.Nodes[node.ParentID]
			if !exists {
				return fmt.Errorf("parent node %d not found for node %d", node.ParentID, node.Index)
			}
			parent.Children = append(parent.Children, node)
		}
	}

	fmt.Printf("Mind map structure built. Root has %d children\n", len(mindmap.Root.Children))

	return nil
}

func (mm *MindMapManager) LoadNodes(mindmapName string) error {
	nodes, err := mm.Store.GetAllNodesForMindMap(mindmapName)
	if err != nil {
		return fmt.Errorf("failed to retrieve nodes: %v", err)
	}

	newMindMap := &MindMap{
		Nodes:    make(map[int]*models.Node),
		MaxIndex: 0,
	}

	// First pass: create all nodes
	for _, node := range nodes {
		newMindMap.Nodes[node.Index] = node
		if node.Index > newMindMap.MaxIndex {
			newMindMap.MaxIndex = node.Index
		}
		node.Children = []*models.Node{}
		fmt.Printf("Loaded node: Index=%d, ParentID=%d, Content='%s', LogicalIndex='%s'\n",
			node.Index, node.ParentID, node.Content, node.LogicalIndex)

		if node.ParentID == -1 {
			newMindMap.Root = node
			fmt.Printf("Root node identified: Index=%d, Content='%s'\n", node.Index, node.Content)
		}
	}

	if newMindMap.Root == nil {
		return fmt.Errorf("root node not found in mindmap '%s'", mindmapName)
	}

	// Second pass: build tree structure
	for _, node := range newMindMap.Nodes {
		if node != newMindMap.Root {
			parent, exists := newMindMap.Nodes[node.ParentID]
			if exists {
				parent.Children = append(parent.Children, node)
				fmt.Printf("Added node %d (%s) as child of node %d (%s)\n",
					node.Index, node.Content, parent.Index, parent.Content)
			} else {
				return fmt.Errorf("parent node %d not found for node %d", node.ParentID, node.Index)
			}
		}
	}

	// Sort children of each node based on LogicalIndex
	var sortNodeChildren func(*models.Node)
	sortNodeChildren = func(node *models.Node) {
		sort.Slice(node.Children, func(i, j int) bool {
			return node.Children[i].LogicalIndex < node.Children[j].LogicalIndex
		})
		for _, child := range node.Children {
			sortNodeChildren(child)
		}
	}
	sortNodeChildren(newMindMap.Root)

	fmt.Printf("Mind map structure built. Root has %d children\n", len(newMindMap.Root.Children))

	// Update the MindMaps map with the new mindmap
	mm.MindMaps[mindmapName] = newMindMap

	return nil
}

func (mm *MindMapManager) debugPrintTree(node *models.Node, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Printf("%sNode: Index=%d, Content='%s', LogicalIndex='%s', Children=%d\n",
		indent, node.Index, node.Content, node.LogicalIndex, len(node.Children))
	for _, child := range node.Children {
		mm.debugPrintTree(child, depth+1)
	}
}

func (mm *MindMapManager) validateAndUpdateLogicalIndices(node *models.Node, parentIndex string) {
	for i, child := range node.Children {
		expectedIndex := fmt.Sprintf("%s%d", parentIndex, i+1)
		if child.LogicalIndex != expectedIndex {
			fmt.Printf("Correcting logical index for node %d: %s -> %s\n",
				child.Index, child.LogicalIndex, expectedIndex)
			child.LogicalIndex = expectedIndex
			// Update the database with the corrected logical index
			err := mm.Store.UpdateNodeOrder(mm.CurrentMindMap.Root.Content, child.Index, child.LogicalIndex)
			if err != nil {
				fmt.Printf("Error updating logical index in database: %v\n", err)
			}
		}
		mm.validateAndUpdateLogicalIndices(child, child.LogicalIndex+".")
	}
}

func (mm *MindMap) assignLogicalIndex(node *models.Node, prefix string) {
	if node == mm.Root {
		node.LogicalIndex = "0"
		prefix = ""
	}

	for i, child := range node.Children {
		child.LogicalIndex = fmt.Sprintf("%s%d", prefix, i+1)
		fmt.Printf("Assigned LogicalIndex %s to node %s\n", child.LogicalIndex, child.Content)
		mm.assignLogicalIndex(child, child.LogicalIndex+".")
	}
}

func (mm *MindMapManager) findNodeByLogicalIndex(logicalIndex string) *models.Node {
	if mm.CurrentMindMap == nil {
		return nil
	}

	if logicalIndex == "0" {
		return mm.CurrentMindMap.Root
	}

	parts := strings.Split(logicalIndex, ".")
	currentNode := mm.CurrentMindMap.Root

	for _, part := range parts {
		index, err := strconv.Atoi(part)
		if err != nil || index < 1 || index > len(currentNode.Children) {
			return nil
		}
		currentNode = currentNode.Children[index-1]
	}

	return currentNode
}

func (mm *MindMapManager) Show(logicalIndex string, showIndex bool) error {
	if err := mm.ensureCurrentMindMap(); err != nil {
		return err
	}

	if mm.CurrentMindMap.Root == nil {
		return fmt.Errorf("current mindmap is empty or not properly initialized")
	}

	var node *models.Node
	if logicalIndex == "" {
		node = mm.CurrentMindMap.Root
		fmt.Printf("Showing entire mind map from root: Index=%d, Content='%s'\n", node.Index, node.Content)
	} else {
		node = mm.findNodeByLogicalIndex(logicalIndex)
		if node == nil {
			return fmt.Errorf("node not found with logical index: %s", logicalIndex)
		}
		fmt.Printf("Showing subtree from node: Index=%d, Content='%s', LogicalIndex='%s'\n", node.Index, node.Content, node.LogicalIndex)
	}

	fmt.Println("Mind Map Structure:")
	mm.visualize(node, "", true, showIndex)

	return nil
}

func (mm *MindMapManager) checkParentChildRelationships() {
	for _, node := range mm.CurrentMindMap.Nodes {
		if node != mm.CurrentMindMap.Root {
			parent, exists := mm.CurrentMindMap.Nodes[node.ParentID]
			if !exists {
				fmt.Printf("WARNING: Node %d (%s) has non-existent parent ID %d\n",
					node.Index, node.Content, node.ParentID)
			} else {
				found := false
				for _, child := range parent.Children {
					if child == node {
						found = true
						break
					}
				}
				if !found {
					fmt.Printf("WARNING: Node %d (%s) is not in the children list of its parent %d (%s)\n",
						node.Index, node.Content, parent.Index, parent.Content)
				}
			}
		}
	}
}

func (mm *MindMapManager) visualize(node *models.Node, prefix string, isLast bool, showIndex bool) {
	if node == nil {
		fmt.Println("DEBUG: WARNING: Attempted to visualize a nil node")
		return
	}

	fmt.Printf("DEBUG: Visualizing node: Index=%d, ParentID=%d, Content='%s', LogicalIndex='%s', Children=%d\n",
		node.Index, node.ParentID, node.Content, node.LogicalIndex, len(node.Children))

	var line strings.Builder

	if node == mm.CurrentMindMap.Root {
		line.WriteString(fmt.Sprintf("%s %s", node.LogicalIndex, node.Content))
	} else {
		if isLast {
			line.WriteString(fmt.Sprintf("%s└── ", prefix))
			prefix += "    "
		} else {
			line.WriteString(fmt.Sprintf("%s├── ", prefix))
			prefix += "│   "
		}
		line.WriteString(fmt.Sprintf("%s %s", node.LogicalIndex, node.Content))
	}

	if showIndex {
		line.WriteString(fmt.Sprintf(" [%d]", node.Index))
	}

	// Add extra fields
	if len(node.Extra) > 0 {
		var extraFields []string
		for k, v := range node.Extra {
			extraFields = append(extraFields, fmt.Sprintf("%s:%s", k, v))
		}
		sort.Strings(extraFields) // Sort extra fields for consistent output
		line.WriteString(" " + strings.Join(extraFields, ", "))
	}

	fmt.Println(line.String())

	// Sort children before visualizing
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].LogicalIndex < node.Children[j].LogicalIndex
	})

	for i, child := range node.Children {
		mm.visualize(child, prefix, i == len(node.Children)-1, showIndex)
	}
}
