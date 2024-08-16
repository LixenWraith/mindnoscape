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
	MindmapReset()
	MindmapList() ([]storage.MindmapInfo, error)
	NodeLoad(name string) error
	MindmapView(logicalIndex string, showIndex bool) ([]string, error)
	MindmapPermission(name string, username string, setPublic ...bool) (bool, error)
	MindmapExport(filename, format string) error
	MindmapImport(filename, format string) error
}

type MindmapManager struct {
	MindmapStore   storage.MindmapStore
	NodeStore      storage.NodeStore
	Mindmaps       map[string]*models.Mindmap
	CurrentMindmap *models.Mindmap
	CurrentUser    string
	NodeManager    *NodeManager
}

func NewMindmapManager(mindmapStore storage.MindmapStore, nodeStore storage.NodeStore, username string) (*MindmapManager, error) {
	mm := &MindmapManager{
		MindmapStore: mindmapStore,
		NodeStore:    nodeStore,
		Mindmaps:     make(map[string]*models.Mindmap),
		CurrentUser:  username,
	}

	mm.NodeManager = NewNodeManager(mm)

	// Load existing mindmaps for the user
	mindmaps, err := mindmapStore.MindmapGetAll(username)
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

func (mm *MindmapManager) MindmapAdd(name string, isPublic bool) error {
	if mm.CurrentUser == "" {
		return fmt.Errorf("no user selected, please select a user first")
	}

	if _, exists := mm.Mindmaps[name]; exists {
		return fmt.Errorf("mindmap '%s' already exists", name)
	}

	// Add to storage and get the new MindmapID
	mindmapID, err := mm.MindmapStore.MindmapAdd(name, mm.CurrentUser, isPublic)
	if err != nil {
		return fmt.Errorf("failed to add mindmap to storage: %w", err)
	}

	newMindmap := models.NewMindmap(mindmapID, name, mm.CurrentUser, isPublic)

	// Create root node
	root := models.NewNode(0, name, mindmapID)
	root.ParentID = -1
	root.LogicalIndex = "0"
	newMindmap.Root = root
	newMindmap.Nodes[0] = root

	if err := mm.NodeStore.NodeAdd(name, mm.CurrentUser, -1, root.Content, root.Extra, root.LogicalIndex); err != nil {
		return fmt.Errorf("failed to add root node: %w", err)
	}

	mm.Mindmaps[name] = newMindmap

	return nil
}

func (mm *MindmapManager) MindmapDelete(name string) error {
	if mm.CurrentUser == "" {
		return fmt.Errorf("no user selected, please select a user first")
	}

	// Check if the mindmap exists and the user has permission to delete it
	mindmapInfo, err := mm.MindmapStore.MindmapGet(name)
	if err != nil {
		return fmt.Errorf("failed to get mindmap info: %w", err)
	}

	if mindmapInfo.Owner != mm.CurrentUser {
		return fmt.Errorf("user '%s' does not have permission to delete mindmap '%s'", mm.CurrentUser, name)
	}

	// Delete the mindmap
	err = mm.MindmapStore.MindmapDelete(name, mm.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to delete mindmap from storage: %w", err)
	}

	// Remove the mindmap from the in-memory map
	delete(mm.Mindmaps, name)

	// If the deleted mindmap was the current mindmap, reset the current mindmap
	if mm.CurrentMindmap != nil && mm.CurrentMindmap.Name == name {
		mm.CurrentMindmap = nil
	}

	return nil
}

func (mm *MindmapManager) MindmapSelect(name string) error {
	if mm.CurrentUser == "" {
		return fmt.Errorf("no user selected, please select a user first")
	}

	if name == "" {
		mm.MindmapReset()
		return nil
	}

	// Load mindmap details
	mindmapInfo, err := mm.MindmapStore.MindmapGet(name)
	if err != nil {
		return fmt.Errorf("failed to get mindmap info: %w", err)
	}

	// Check if the user has permission to access the mindmap
	if !mindmapInfo.IsPublic && mindmapInfo.Owner != mm.CurrentUser {
		return fmt.Errorf("user %s does not have permission to access mindmap '%s'", mm.CurrentUser, name)
	}

	// Create or update the mindmap in the Mindmaps map
	mindmap, exists := mm.Mindmaps[name]
	if !exists {
		mindmap = &models.Mindmap{
			Name:     mindmapInfo.Name,
			IsPublic: mindmapInfo.IsPublic,
			Owner:    mindmapInfo.Owner,
			Root:     &models.Node{},
		}
		mm.Mindmaps[name] = mindmap
	} else {
		mindmap.IsPublic = mindmapInfo.IsPublic
		mindmap.Owner = mindmapInfo.Owner
	}

	// Load nodes for the selected mindmap
	nodes, err := mm.NodeStore.NodeGetAll(name, mm.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to load nodes for mindmap '%s': %w", name, err)
	}

	// Build the mindmap structure
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
	mindmap.Nodes = nodeMap

	mm.CurrentMindmap = mindmap
	mm.NodeManager.NodeReset()

	return nil
}

func (mm *MindmapManager) MindmapReset() {
	mm.Mindmaps = make(map[string]*models.Mindmap)
	mm.CurrentMindmap = nil
	mm.NodeManager.NodeReset()
}

func (mm *MindmapManager) MindmapList() ([]storage.MindmapInfo, error) {
	if mm.CurrentUser == "" {
		return nil, fmt.Errorf("no user selected, please select a user first")
	}
	mindmaps, err := mm.MindmapStore.MindmapGetAll(mm.CurrentUser)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve mindmaps: %w", err)
	}

	// Filter out private mindmaps that don't belong to the current user
	var filteredMindmaps []storage.MindmapInfo
	for _, m := range mindmaps {
		if m.IsPublic || m.Owner == mm.CurrentUser {
			filteredMindmaps = append(filteredMindmaps, m)
		}
	}

	return filteredMindmaps, nil
}

func (mm *MindmapManager) MindmapView(logicalIndex string, showIndex bool) ([]string, error) {
	if mm.CurrentMindmap == nil {
		return nil, fmt.Errorf("no mindmap selected")
	}

	var node *models.Node
	var err error

	if logicalIndex == "" {
		node = mm.CurrentMindmap.Root
	} else {
		node, err = mm.findNodeByLogicalIndex(logicalIndex)
		if err != nil {
			return nil, err
		}
	}

	return mm.visualizeMindmap(node, "", true, showIndex)
}

func (mm *MindmapManager) MindmapPermission(name string, username string, setPublic ...bool) (bool, error) {
	return mm.MindmapStore.MindmapPermission(name, username, setPublic...)
}

func (mm *MindmapManager) MindmapExport(filename, format string) error {
	if mm.CurrentMindmap == nil {
		return fmt.Errorf("no mindmap selected")
	}

	err := storage.FileExport(mm.MindmapStore, mm.NodeStore, mm.CurrentMindmap.Name, mm.CurrentUser, filename, format)
	if err != nil {
		return fmt.Errorf("failed to export mindmap: %w", err)
	}

	return nil
}

func (mm *MindmapManager) MindmapImport(filename, format string) error {
	err := storage.FileImport(mm.MindmapStore, mm.NodeStore, mm.CurrentUser, filename, format)
	if err != nil {
		return fmt.Errorf("failed to import mindmap: %w", err)
	}

	// Get the name of the imported mindmap (which is the root node's content)
	nodes, err := mm.NodeStore.NodeGetAll("", mm.CurrentUser)
	if err != nil || len(nodes) == 0 {
		return fmt.Errorf("failed to retrieve imported mindmap data: %w", err)
	}
	importedMindmapName := nodes[0].Content // The root node's content is the mindmap name

	// Switch to the newly imported mindmap
	return mm.MindmapSelect(importedMindmapName)
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
