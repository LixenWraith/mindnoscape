package mindmap

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	// "sync"

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
	// mutex          sync.RWMutex
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

func (mm *MindMapManager) mindMapExists(name string) bool {
	_, exists := mm.MindMaps[name]
	return exists
}

func (mm *MindMapManager) CreateNewMindMap(name string) error {
	// mm.mutex.Lock()
	// defer mm.mutex.Unlock()

	if mm.mindMapExists(name) {
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
	if !mm.mindMapExists(name) {
		return fmt.Errorf("mindmap '%s' does not exist", name)
	}

	mm.CurrentMindMap = mm.MindMaps[name]

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

func (mm *MindMapManager) buildTreeFromNodes(mindmap *MindMap, nodes []*models.Node) error {
	for _, node := range nodes {
		mindmap.Nodes[node.Index] = node
		if node.Index > mindmap.MaxIndex {
			mindmap.MaxIndex = node.Index
		}
		node.Children = []*models.Node{}

		if node.ParentID == -1 {
			mindmap.Root = node
		}
	}

	if mindmap.Root == nil {
		return fmt.Errorf("root node not found")
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

	return nil
}

func (mm *MindMapManager) loadNodesForMindMap(name string) error {
	nodes, err := mm.Store.GetAllNodesForMindMap(name)
	if err != nil {
		return fmt.Errorf("failed to retrieve nodes for mindmap '%s': %v", name, err)
	}

	mindmap := mm.MindMaps[name]
	mindmap.Nodes = make(map[int]*models.Node)
	mindmap.MaxIndex = 0

	err = mm.buildTreeFromNodes(mindmap, nodes)
	if err != nil {
		return fmt.Errorf("failed to build tree structure: %v", err)
	}

	return nil
}

func (mm *MindMapManager) LoadNodes(mindmapName string) error {
	if !mm.mindMapExists(mindmapName) {
		return fmt.Errorf("mindmap '%s' does not exist", mindmapName)
	}

	nodes, err := mm.Store.GetAllNodesForMindMap(mindmapName)
	if err != nil {
		return fmt.Errorf("failed to retrieve nodes: %v", err)
	}

	newMindMap := &MindMap{
		Nodes:    make(map[int]*models.Node),
		MaxIndex: 0,
	}

	err = mm.buildTreeFromNodes(newMindMap, nodes)
	if err != nil {
		return fmt.Errorf("failed to build tree structure: %v", err)
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

	// Update the MindMaps map with the new mindmap
	mm.MindMaps[mindmapName] = newMindMap

	return nil
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

func (mm *MindMapManager) findNodeByLogicalIndex(logicalIndex string) (*models.Node, error) {
	if mm.CurrentMindMap == nil {
		return nil, fmt.Errorf("no current mindmap selected")
	}

	if logicalIndex == "0" {
		return mm.CurrentMindMap.Root, nil
	}

	parts := strings.Split(logicalIndex, ".")
	currentNode := mm.CurrentMindMap.Root

	for i, part := range parts {
		index, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid logical index part '%s': not a number", part)
		}
		if index < 1 || index > len(currentNode.Children) {
			return nil, fmt.Errorf("invalid logical index: part %d (%s) is out of range", i+1, part)
		}
		currentNode = currentNode.Children[index-1]
	}

	return currentNode, nil
}

func (mm *MindMapManager) Show(logicalIndex string, showIndex bool) error {
	if mm.CurrentMindMap == nil {

		return fmt.Errorf("no mindmap selected, use 'switch' command to select a mindmap")
	}

	if mm.CurrentMindMap.Root == nil {
		return fmt.Errorf("current mindmap is empty or not properly initialized")
	}

	var node *models.Node

	if logicalIndex == "" {
		node = mm.CurrentMindMap.Root
		fmt.Printf("Showing entire mind map from root: Index=%d, Content='%s'\n", node.Index, node.Content)
	} else {
		var err error
		node, err = mm.findNodeByLogicalIndex(logicalIndex)
		if err != nil {
			return fmt.Errorf("failed to find node: %v", err)
		}
		fmt.Printf("Showing subtree from node: Index=%d, Content='%s', LogicalIndex='%s'\n", node.Index, node.Content, node.LogicalIndex)
	}

	fmt.Println("Mind Map Structure:")
	if err := mm.visualize(node, "", true, showIndex); err != nil {
		return fmt.Errorf("failed to visualize mind map: %v", err)
	}

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

func (mm *MindMapManager) visualize(node *models.Node, prefix string, isLast bool, showIndex bool) error {
	if node == nil {
		return fmt.Errorf("attempted to visualize a nil node")
	}

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
		err := mm.visualize(child, prefix, i == len(node.Children)-1, showIndex)
		if err != nil {
			return fmt.Errorf("error visualizing child node: %v", err)
		}
	}

	return nil
}
