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

const (
	ColorYellow    = "{{yellow}}"
	ColorOrange    = "{{orange}}"
	ColorDarkBrown = "{{darkbrown}}"
	ColorDefault   = "{{default}}"
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
	CurrentUser    string
	history        []Operation
	historyIndex   int
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

func (mm *MindMapManager) ChangeUser(username string) error {
	// Load mindmaps for the new user
	mindmaps, err := mm.Store.GetAllMindMaps(username)
	if err != nil {
		return fmt.Errorf("failed to load mindmaps for user %s: %v", username, err)
	}

	// Update MindMaps map
	for _, mindmap := range mindmaps {
		if _, exists := mm.MindMaps[mindmap.Name]; !exists {
			mm.MindMaps[mindmap.Name] = &MindMap{
				Nodes: make(map[int]*models.Node),
			}
		}
	}

	// Remove any mindmaps not in the new user's list
	for name := range mm.MindMaps {
		found := false
		for _, mindmap := range mindmaps {
			if mindmap.Name == name {
				found = true
				break
			}
		}
		if !found {
			delete(mm.MindMaps, name)
		}
	}

	mm.CurrentMindMap = nil
	mm.CurrentUser = username
	mm.ClearOperationHistory()

	return nil
}

func NewMindMapManager(store storage.Store, username string) (*MindMapManager, error) {
	mm := &MindMapManager{
		Store:        store,
		MindMaps:     make(map[string]*MindMap),
		CurrentUser:  username,
		history:      []Operation{},
		historyIndex: -1,
	}

	// Load existing mindmaps for the user
	mindmaps, err := store.GetAllMindMaps(username)
	if err != nil {
		return nil, fmt.Errorf("failed to load mindmaps: %v", err)
	}

	for _, mindmap := range mindmaps {
		mm.MindMaps[mindmap.Name] = &MindMap{
			Nodes: make(map[int]*models.Node),
		}
	}

	return mm, nil
}

func (mm *MindMapManager) mindMapExists(name string) bool {
	_, exists := mm.MindMaps[name]
	return exists
}

func (mm *MindMapManager) ListMindMaps() ([]storage.MindMapInfo, error) {
	allMindmaps, err := mm.Store.GetAllMindMaps(mm.CurrentUser)
	if err != nil {
		return nil, err
	}

	var existingMindmaps []storage.MindMapInfo
	for _, mindmap := range allMindmaps {
		if _, exists := mm.MindMaps[mindmap.Name]; exists {
			existingMindmaps = append(existingMindmaps, mindmap)
		}
	}

	return existingMindmaps, nil
}

func (mm *MindMapManager) CreateNewMindMap(name string, isPublic bool) error {
	if mm.mindMapExists(name) {
		return fmt.Errorf("mindmap '%s' already exists", name)
	}

	// Add to storage and get the new MindmapID
	mindmapID, err := mm.Store.AddMindMap(name, mm.CurrentUser, isPublic)
	if err != nil {
		return fmt.Errorf("failed to add mindmap to storage: %v", err)
	}

	newMindMap := &MindMap{
		Nodes: make(map[int]*models.Node),
	}

	// Create root node with the correct MindmapID
	root := models.NewNode(0, name, mindmapID)
	root.ParentID = -1
	root.LogicalIndex = "0"
	newMindMap.Root = root
	newMindMap.Nodes[0] = root

	if err := mm.Store.AddNode(name, mm.CurrentUser, -1, root.Content, root.Extra, root.LogicalIndex); err != nil {
		return fmt.Errorf("failed to add root node: %v", err)
	}

	mm.MindMaps[name] = newMindMap
	mm.CurrentMindMap = newMindMap

	return nil
}

func (mm *MindMapManager) RemoveMindMap(name string) {
	delete(mm.MindMaps, name)
}

func (mm *MindMapManager) SwitchMindMap(name string) error {
	if !mm.mindMapExists(name) {
		return fmt.Errorf("mindmap '%s' does not exist", name)
	}

	// Check if the user has permission to access this mindmap
	hasPermission, err := mm.Store.HasMindMapPermission(name, mm.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to check mindmap permissions: %v", err)
	}
	if !hasPermission {
		return fmt.Errorf("user %s does not have permission to access mindmap '%s'", mm.CurrentUser, name)
	}

	// Load nodes for the switched mindmap
	if err := mm.loadNodesForMindMap(name); err != nil {
		return fmt.Errorf("failed to load nodes for mindmap '%s': %v", name, err)
	}

	mm.CurrentMindMap = mm.MindMaps[name]

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
	nodes, err := mm.Store.GetAllNodesForMindMap(name, mm.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to retrieve nodes for mindmap '%s': %v", name, err)
	}

	mindmap := &MindMap{
		Nodes:    make(map[int]*models.Node),
		MaxIndex: 0,
	}

	err = mm.buildTreeFromNodes(mindmap, nodes)
	if err != nil {
		return fmt.Errorf("failed to build tree structure: %v", err)
	}

	mm.MindMaps[name] = mindmap
	mm.ClearOperationHistory()

	return nil
}

func (mm *MindMapManager) LoadNodes(mindmapName string) error {
	if !mm.mindMapExists(mindmapName) {
		return fmt.Errorf("mindmap '%s' does not exist", mindmapName)
	}

	nodes, err := mm.Store.GetAllNodesForMindMap(mindmapName, mm.CurrentUser)
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

func (mm *MindMap) assignLogicalIndex(node *models.Node, prefix string) {
	if node == mm.Root {
		node.LogicalIndex = "0"
		prefix = ""
	}

	for i, child := range node.Children {
		child.LogicalIndex = fmt.Sprintf("%s%d", prefix, i+1)
		mm.assignLogicalIndex(child, child.LogicalIndex+".")
	}
}

func (mm *MindMapManager) findNodeByLogicalIndex(logicalIndex string) (*models.Node, error) {
	if err := mm.ensureCurrentMindMap(); err != nil {
		return nil, err
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

func (mm *MindMapManager) Show(logicalIndex string, showIndex bool) ([]string, error) {
	var output []string

	if err := mm.ensureCurrentMindMap(); err != nil {
		return nil, err
	}

	if mm.CurrentMindMap.Root == nil {
		return nil, fmt.Errorf("current mindmap is empty or not properly initialized")
	}

	var node *models.Node

	if logicalIndex == "" {
		node = mm.CurrentMindMap.Root
	} else {
		var err error
		node, err = mm.findNodeByLogicalIndex(logicalIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to find node: %v", err)
		}
	}

	output = append(output, "Mind Map Structure:")
	visualOutput, err := mm.visualize(node, "", true, showIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to visualize mind map: %v", err)
	}
	output = append(output, visualOutput...)

	return output, nil
}

func (mm *MindMapManager) visualize(node *models.Node, prefix string, isLast bool, showIndex bool) ([]string, error) {
	var output []string

	if node == nil {
		return nil, fmt.Errorf("attempted to visualize a nil node")
	}

	var line strings.Builder

	if node == mm.CurrentMindMap.Root {
		line.WriteString(fmt.Sprintf("%s%s%s %s", ColorYellow, node.LogicalIndex, ColorDefault, node.Content))
	} else {
		line.WriteString(prefix)
		if isLast {
			line.WriteString(fmt.Sprintf("%s└── %s", ColorDarkBrown, ColorDefault))
			prefix += fmt.Sprintf("%s    ", ColorDarkBrown)
		} else {
			line.WriteString(fmt.Sprintf("%s├── %s", ColorDarkBrown, ColorDefault))
			prefix += fmt.Sprintf("%s│   ", ColorDarkBrown)
		}
		line.WriteString(fmt.Sprintf("%s%s%s %s", ColorYellow, node.LogicalIndex, ColorDefault, node.Content))
	}

	if showIndex {
		line.WriteString(fmt.Sprintf(" %s[%d]%s", ColorOrange, node.Index, ColorDefault))
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

	output = append(output, line.String())

	// Sort children based on their logical index
	sort.Slice(node.Children, func(i, j int) bool {
		return compareLogicalIndexes(node.Children[i].LogicalIndex, node.Children[j].LogicalIndex)
	})

	for i, child := range node.Children {
		childOutput, err := mm.visualize(child, prefix, i == len(node.Children)-1, showIndex)
		if err != nil {
			return nil, fmt.Errorf("error visualizing child node: %v", err)
		}
		output = append(output, childOutput...)
	}

	return output, nil
}

// Helper function to compare logical indexes
func compareLogicalIndexes(a, b string) bool {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		aNum, _ := strconv.Atoi(aParts[i])
		bNum, _ := strconv.Atoi(bParts[i])
		if aNum != bNum {
			return aNum < bNum
		}
	}

	return len(aParts) < len(bParts)
}

func (mm *MindMapManager) Undo() error {
	if mm.historyIndex < 0 {
		return fmt.Errorf("nothing to undo")
	}

	op := mm.history[mm.historyIndex]

	var err error
	switch op.Type {
	case OpAdd:
		err = mm.DeleteNode(strconv.Itoa(op.AffectedNode.Index), true, false)
	case OpDelete:
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
	return nil
}

func (mm *MindMapManager) Redo() error {
	if mm.historyIndex >= len(mm.history)-1 {
		return fmt.Errorf("nothing to redo")
	}

	op := mm.history[mm.historyIndex+1]

	var err error
	switch op.Type {
	case OpAdd:
		err = mm.AddNode(strconv.Itoa(op.AffectedNode.ParentID), op.NewContent, op.NewExtra, true, op.AffectedNode.Index, false)
	case OpDelete:
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
}

func (mm *MindMapManager) ClearOperationHistory() {
	mm.history = []Operation{}
	mm.historyIndex = -1
}
