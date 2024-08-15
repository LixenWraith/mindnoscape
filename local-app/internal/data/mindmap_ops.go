package data

import (
	"fmt"
	"strconv"
	"strings"

	"mindnoscape/local-app/internal/models"
	"mindnoscape/local-app/internal/storage"
)

// MindmapOperations defines the interface for data-related operations
type MindmapOperations interface {
	MindmapAdd(name string, isPublic bool) error
	MindmapDelete(name string) error
	MindmapSelect(name string) error
	MindmapList() ([]storage.MindmapInfo, error)
	NodeLoad(name string) error
	MindmapView(logicalIndex string, showIndex bool) ([]string, error)
	MindmapPermission(name string, isPublic bool) error
	MindmapSave(filename, format string) error
	MindmapLoad(filename, format string) error
}

type MindmapManager struct {
	Store          storage.Store
	Mindmaps       map[string]*models.Mindmap
	CurrentMindmap *models.Mindmap
	CurrentUser    string
	HistoryManager *HistoryManager
	NodeManager    *NodeManager
}

func NewMindmapManager(store storage.Store, username string) (*MindmapManager, error) {
	mm := &MindmapManager{
		Store:       store,
		Mindmaps:    make(map[string]*models.Mindmap),
		CurrentUser: username,
	}

	mm.NodeManager = NewNodeManager(mm)
	mm.HistoryManager = NewHistoryManager(mm)

	// Load existing mindmaps for the user
	mindmaps, err := store.MindmapGetAll(username)
	if err != nil {
		return nil, fmt.Errorf("failed to load mindmaps: %w", err)
	}

	for _, mindmapInfo := range mindmaps {
		mm.Mindmaps[mindmapInfo.Name] = &models.Mindmap{
			Name:     mindmapInfo.Name,
			IsPublic: mindmapInfo.IsPublic,
			Owner:    mindmapInfo.Owner,
			Root:     &models.Node{},
		}
	}

	return mm, nil
}

func (mm *MindmapManager) UserSelect(username string) error {
	mm.CurrentUser = username

	// Clear current mindmaps
	mm.Mindmaps = make(map[string]*models.Mindmap)
	mm.CurrentMindmap = nil

	// Load mindmaps for the new user
	mindmaps, err := mm.Store.MindmapGetAll(username)
	if err != nil {
		return fmt.Errorf("failed to load mindmaps for user %s: %w", username, err)
	}

	for _, mindmapInfo := range mindmaps {
		mm.Mindmaps[mindmapInfo.Name] = &models.Mindmap{
			Name:     mindmapInfo.Name,
			IsPublic: mindmapInfo.IsPublic,
			Owner:    mindmapInfo.Owner,
			Root:     &models.Node{},
			Nodes:    make(map[int]*models.Node),
			MaxIndex: 0,
		}
	}

	return nil
}

func (mm *MindmapManager) MindmapAdd(name string, isPublic bool) error {
	if mm.Mindmaps == nil {
		mm.Mindmaps = make(map[string]*models.Mindmap)
	}

	if _, exists := mm.Mindmaps[name]; exists {
		return fmt.Errorf("data '%s' already exists", name)
	}

	// Add to storage and get the new MindmapID
	mindmapID, err := mm.Store.MindmapAdd(name, mm.CurrentUser, isPublic)
	if err != nil {
		return fmt.Errorf("failed to add data to storage: %w", err)
	}

	newMindmap := models.NewMindmap(mindmapID, name, mm.CurrentUser, isPublic)

	// Create root node
	root := models.NewNode(0, name, mindmapID)
	root.ParentID = -1
	root.LogicalIndex = "0"
	newMindmap.Root = root
	newMindmap.Nodes[0] = root

	if err := mm.Store.NodeAdd(name, mm.CurrentUser, -1, root.Content, root.Extra, root.LogicalIndex); err != nil {
		return fmt.Errorf("failed to add root node: %w", err)
	}

	mm.Mindmaps[name] = newMindmap
	mm.CurrentMindmap = newMindmap

	return nil
}

func (mm *MindmapManager) MindmapDelete(name string) error {
	if _, exists := mm.Mindmaps[name]; !exists {
		return fmt.Errorf("data '%s' does not exist", name)
	}

	err := mm.Store.MindmapDelete(name, mm.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to delete data from storage: %w", err)
	}

	delete(mm.Mindmaps, name)

	if mm.CurrentMindmap != nil && mm.CurrentMindmap.Name == name {
		mm.CurrentMindmap = nil
	}

	return nil
}

func (mm *MindmapManager) MindmapSelect(name string) error {
	mindmap, exists := mm.Mindmaps[name]
	if !exists {
		return fmt.Errorf("data '%s' does not exist", name)
	}

	// Check if the user has permission to access this data
	hasPermission, err := mm.Store.MindmapPermission(name, mm.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to check data permissions: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("user %s does not have permission to access data '%s'", mm.CurrentUser, name)
	}

	// Load nodes for the switched data
	nodes, err := mm.Store.NodeGetAll(name, mm.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to load nodes for data '%s': %w", name, err)
	}

	// Build the data structure
	nodeMap := make(map[int]*models.Node)
	for _, node := range nodes {
		nodeMap[node.Index] = node
	}

	for _, node := range nodes {
		if node.ParentID != -1 {
			parent := nodeMap[node.ParentID]
			if parent != nil {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	mindmap.Root = nodeMap[0] // Assuming the root node always has index 0
	mm.CurrentMindmap = mindmap

	return nil
}

func (mm *MindmapManager) MindmapList() ([]storage.MindmapInfo, error) {
	return mm.Store.MindmapGetAll(mm.CurrentUser)
}

func (mm *MindmapManager) NodeLoad(mindmapName string) error {
	nodes, err := mm.Store.NodeGetAll(mindmapName, mm.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to load nodes for data '%s': %w", mindmapName, err)
	}

	mindmap, exists := mm.Mindmaps[mindmapName]
	if !exists {
		return fmt.Errorf("data '%s' does not exist", mindmapName)
	}

	mindmap.Nodes = make(map[int]*models.Node)
	for _, node := range nodes {
		mindmap.Nodes[node.Index] = node
		if node.Index > mindmap.MaxIndex {
			mindmap.MaxIndex = node.Index
		}
	}

	// Build the tree structure
	for _, node := range nodes {
		if node.ParentID != -1 {
			parent := mindmap.Nodes[node.ParentID]
			if parent != nil {
				parent.Children = append(parent.Children, node)
			}
		} else {
			mindmap.Root = node
		}
	}

	return nil
}

func (mm *MindmapManager) MindmapView(logicalIndex string, showIndex bool) ([]string, error) {
	if mm.CurrentMindmap == nil {
		return nil, fmt.Errorf("no data selected")
	}

	var node *models.Node
	var err error

	if logicalIndex == "" {
		node = mm.CurrentMindmap.Root
	} else {
		node, err = mm.findNodeByLogicalIndex(logicalIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to find node: %w", err)
		}
	}

	output := []string{"Mind Map Structure:"}
	visualOutput, err := mm.visualizeMindmap(node, "", true, showIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to visualize mind map: %w", err)
	}
	output = append(output, visualOutput...)

	return output, nil
}

func (mm *MindmapManager) MindmapPermission(name string, username string, setPublic ...bool) (bool, error) {
	mindmap, exists := mm.Mindmaps[name]
	if !exists {
		return false, fmt.Errorf("mindmap '%s' does not exist", name)
	}

	isOwner := mindmap.Owner == username

	// If setPublic is provided, attempt to modify the mindmap's public status
	if len(setPublic) > 0 {
		if !isOwner {
			return false, fmt.Errorf("user '%s' does not have permission to modify mindmap '%s'", username, name)
		}

		newPublicStatus := setPublic[0]
		if mindmap.IsPublic != newPublicStatus {
			hasPermission, err := mm.Store.MindmapPermission(name, username, newPublicStatus)
			if err != nil {
				return false, fmt.Errorf("failed to update mindmap access: %w", err)
			}
			mindmap.IsPublic = newPublicStatus
			return hasPermission, nil
		}
	}

	// Check permission without modifying
	hasPermission, err := mm.Store.MindmapPermission(name, username)
	if err != nil {
		return false, fmt.Errorf("failed to check mindmap permission: %w", err)
	}

	return hasPermission, nil
}

func (mm *MindmapManager) MindmapExport(filename, format string) error {
	if mm.CurrentMindmap == nil {
		return fmt.Errorf("no data selected")
	}

	err := storage.SaveToFile(mm.Store, mm.CurrentMindmap.Name, mm.CurrentUser, filename, format)
	if err != nil {
		return fmt.Errorf("failed to save data: %w", err)
	}

	return nil
}

func (mm *MindmapManager) MindmapImport(filename, format string) error {
	tempRoot, err := storage.ImportFromFile(filename, format)
	if err != nil {
		return fmt.Errorf("failed to import data from file: %w", err)
	}

	mindmapName := tempRoot.Content
	exists, err := mm.Store.MindmapExists(mindmapName, mm.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to check if data exists: %w", err)
	}

	if exists {
		// Delete existing data
		err = mm.MindmapDelete(mindmapName)
		if err != nil {
			return fmt.Errorf("failed to delete existing data: %w", err)
		}
	}

	// Create new data
	err = mm.MindmapAdd(mindmapName, false) // Set isPublic to false by default
	if err != nil {
		return fmt.Errorf("failed to create new data: %w", err)
	}

	// Load nodes into the new data
	err = storage.LoadFromFile(mm.Store, mindmapName, mm.CurrentUser, filename, format)
	if err != nil {
		return fmt.Errorf("failed to load nodes for data '%s': %w", mindmapName, err)
	}

	// Switch to the newly loaded data
	return mm.MindmapSelect(mindmapName)
}

// Helper functions

func (mm *MindmapManager) findNodeByLogicalIndex(logicalIndex string) (*models.Node, error) {
	if logicalIndex == "0" {
		return mm.CurrentMindmap.Root, nil
	}

	parts := strings.Split(logicalIndex, ".")
	currentNode := mm.CurrentMindmap.Root

	for _, part := range parts {
		index, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid logical index part '%s': %w", part, err)
		}
		if index < 1 || index > len(currentNode.Children) {
			return nil, fmt.Errorf("invalid logical index: part %s is out of range", part)
		}
		currentNode = currentNode.Children[index-1]
	}

	return currentNode, nil
}

func (mm *MindmapManager) visualizeMindmap(node *models.Node, prefix string, isLast bool, showIndex bool) ([]string, error) {
	var output []string

	// Current node
	var line strings.Builder
	line.WriteString(prefix)

	if isLast {
		line.WriteString(fmt.Sprintf("%s%s%s", string(ColorDarkBrown), "└── ", string(ColorDarkBrown)))
		prefix += "    "
	} else {
		line.WriteString(fmt.Sprintf("%s%s%s", string(ColorDarkBrown), "├── ", string(ColorDarkBrown)))
		prefix += fmt.Sprintf("%s%s%s", string(ColorDarkBrown), "│   ", string(ColorDarkBrown))
	}

	line.WriteString(fmt.Sprintf("%s%s%s", string(ColorYellow), node.LogicalIndex, string(ColorDefault)))
	line.WriteString(" " + node.Content)

	if showIndex {
		line.WriteString(fmt.Sprintf(" %s[%d]%s", string(ColorOrange), node.Index, string(ColorDefault)))
	}

	// Add extra fields
	if len(node.Extra) > 0 {
		var extraFields []string
		for k, v := range node.Extra {
			extraFields = append(extraFields, fmt.Sprintf("%s:%s", k, v))
		}
		line.WriteString(" " + strings.Join(extraFields, ", "))
	}

	output = append(output, line.String())

	// Children nodes
	for i, child := range node.Children {
		childOutput, err := mm.visualizeMindmap(child, prefix, i == len(node.Children)-1, showIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to visualize child node: %w", err)
		}
		output = append(output, childOutput...)
	}

	return output, nil
}