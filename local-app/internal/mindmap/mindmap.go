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
	history        []Operation
	historyIndex   int
	// mutex          sync.RWMutex
}

type OperationType string

const (
	OpAdd    OperationType = "Add"
	OpDelete OperationType = "Delete"
	OpMove   OperationType = "Move"
	OpInsert OperationType = "Insert"
	OpModify OperationType = "Modify"
)

type NodeInfo struct {
	Index    int
	ParentID int
}

type Operation struct {
	Type         OperationType
	AffectedNode NodeInfo
	OldParentID  int               // Used for Move and Insert
	NewParentID  int               // Used for Move and Insert
	OldContent   string            // Used for Modify
	NewContent   string            // Used for Modify and Add
	OldExtra     map[string]string // Used for Modify
	NewExtra     map[string]string // Used for Modify and Add
	DeletedTree  []*models.Node    // Used for Delete to store the entire deleted subtree
}

func NewMindMapManager(store storage.Store) (*MindMapManager, error) {
	mm := &MindMapManager{
		Store:        store,
		MindMaps:     make(map[string]*MindMap),
		history:      []Operation{},
		historyIndex: -1,
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

	mm.ClearOperationHistory()

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

	mm.ClearOperationHistory()

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

func (mm *MindMapManager) Undo() error {
	mm.printOperationHistory()

	if mm.historyIndex < 0 {
		return fmt.Errorf("nothing to undo")
	}

	op := mm.history[mm.historyIndex]
	fmt.Printf("Debug: Undoing operation type: %s\n", op.Type)

	var err error
	switch op.Type {
	case OpAdd:
		fmt.Printf("Debug: Undoing Add operation. Index: %d\n", op.AffectedNode.Index)
		err = mm.DeleteNode(strconv.Itoa(op.AffectedNode.Index), true, false)
	case OpDelete:
		fmt.Printf("Debug: Undoing Delete operation. Index: %d, Content: %s\n", op.AffectedNode.Index, op.OldContent)
		fmt.Printf("Debug: Undoing Delete operation. Index: %d, Content: %s\n", op.AffectedNode.Index, op.OldContent)
		err = mm.restoreSubtree(op.DeletedTree, false)
		if err != nil {
			break
		}
		// Recalculate logical indexes after restoring the subtree
		err = mm.recalculateLogicalIndices(mm.CurrentMindMap.Root)
	case OpMove, OpInsert:
		err = mm.MoveNode(strconv.Itoa(op.AffectedNode.Index), strconv.Itoa(op.OldParentID), true, false)
	case OpModify:
		err = mm.ModifyNode(strconv.Itoa(op.AffectedNode.Index), op.OldContent, op.OldExtra, true, false)
	}

	if err != nil {
		return fmt.Errorf("failed to undo %s: %v", op.Type, err)
	}

	mm.historyIndex--
	mm.printOperationHistory()
	return nil
}

func (mm *MindMapManager) Redo() error {
	mm.printOperationHistory()

	if mm.historyIndex >= len(mm.history)-1 {
		return fmt.Errorf("nothing to redo")
	}

	op := mm.history[mm.historyIndex+1]
	fmt.Printf("Debug: Redoing operation type: %s\n", op.Type)

	var err error
	switch op.Type {
	case OpAdd:
		fmt.Printf("Debug: Redoing Add operation. ParentID: %d, Content: %s\n", op.AffectedNode.ParentID, op.NewContent)
		err = mm.AddNode(strconv.Itoa(op.AffectedNode.ParentID), op.NewContent, op.NewExtra, true, op.AffectedNode.Index, false)
	case OpDelete:
		fmt.Printf("Debug: Redoing Delete operation. Index: %d\n", op.AffectedNode.Index)
		err = mm.DeleteNode(strconv.Itoa(op.AffectedNode.Index), true, false)
	case OpMove, OpInsert:
		err = mm.MoveNode(strconv.Itoa(op.AffectedNode.Index), strconv.Itoa(op.NewParentID), true, false)
	case OpModify:
		err = mm.ModifyNode(strconv.Itoa(op.AffectedNode.Index), op.NewContent, op.NewExtra, true, false)
	}

	if err != nil {
		return fmt.Errorf("failed to redo %s: %v", op.Type, err)
	}

	mm.historyIndex++
	mm.printOperationHistory()
	return nil
}

func (mm *MindMapManager) restoreSubtree(nodes []*models.Node, addToHistory bool) error {
	for _, node := range nodes {
		// Only add the node if it doesn't already exist
		if existingNode := mm.CurrentMindMap.Nodes[node.Index]; existingNode == nil {
			err := mm.AddNode(strconv.Itoa(node.ParentID), node.Content, node.Extra, true, node.Index, addToHistory)
			if err != nil {
				return fmt.Errorf("failed to restore node %d: %v", node.Index, err)
			}

			// Restore children recursively
			if len(node.Children) > 0 {
				err = mm.restoreSubtree(node.Children, addToHistory)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (mm *MindMapManager) addToHistory(op Operation) {
	// Check if we're adding a new operation (not from undo/redo)
	if mm.historyIndex == len(mm.history)-1 {
		// Remove any forward history
		mm.history = mm.history[:mm.historyIndex+1]

		// Append the new operation
		mm.history = append(mm.history, op)
		mm.historyIndex++
	} else {
		// We're in the middle of the history (after some undos)
		// Clear everything after the current index and add the new operation
		mm.history = append(mm.history[:mm.historyIndex+1], op)
		mm.historyIndex++
	}

	// Print operation history after modifying it
	mm.printOperationHistory()
}

func (mm *MindMapManager) ClearOperationHistory() {
	mm.history = []Operation{}
	mm.historyIndex = -1
}

func (mm *MindMapManager) printOperationHistory() {
	fmt.Printf("Operation History (length: %d, current index: %d):\n", len(mm.history), mm.historyIndex)
	for i, op := range mm.history {
		fmt.Printf("[%d] Type: %s, AffectedNode: {Index: %d, ParentID: %d}, OldContent: %s, NewContent: %s\n",
			i, op.Type, op.AffectedNode.Index, op.AffectedNode.ParentID, op.OldContent, op.NewContent)

		if len(op.OldExtra) > 0 {
			fmt.Printf("    OldExtra: %v\n", op.OldExtra)
		}
		if len(op.NewExtra) > 0 {
			fmt.Printf("    NewExtra: %v\n", op.NewExtra)
		}

		if op.Type == OpDelete && len(op.DeletedTree) > 0 {
			fmt.Printf("    DeletedTree:\n")
			mm.printDeletedSubtree(op.DeletedTree, "    ")
		}

		if op.Type == OpMove || op.Type == OpInsert {
			fmt.Printf("    OldParentID: %d, NewParentID: %d\n", op.OldParentID, op.NewParentID)
		}
	}
}

func (mm *MindMapManager) printDeletedSubtree(nodes []*models.Node, indent string) {
	for _, node := range nodes {
		fmt.Printf("%s- Index: %d, Content: %s, LogicalIndex: %s\n", indent, node.Index, node.Content, node.LogicalIndex)
		if len(node.Extra) > 0 {
			fmt.Printf("%s  Extra: %v\n", indent, node.Extra)
		}
		if len(node.Children) > 0 {
			mm.printDeletedSubtree(node.Children, indent+"  ")
		}
	}
}
