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
	MindmapList() ([]models.MindmapInfo, error)
	NodeLoad(name string) error
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
	root.Index = "0"

	rootID, err := mm.NodeStore.NodeAdd(name, mm.CurrentUser, -1, root.Content, root.Extra, root.Index)
	if err != nil {
		return fmt.Errorf("failed to add root node: %w", err)
	}
	root.ID = rootID
	newMindmap.Root = root
	newMindmap.Nodes[rootID] = root

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

	// Delete the mindmap and all associated data
	err = mm.MindmapStore.MindmapDelete(name, mm.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to delete mindmap from storage: %w", err)
	}

	// Remove the mindmap from the in-memory map
	delete(mm.Mindmaps, name)

	// If the deleted mindmap was the current mindmap, reset the current mindmap
	if mm.CurrentMindmap != nil && mm.CurrentMindmap.Name == name {
		mm.CurrentMindmap = nil
		mm.NodeManager.NodeReset()
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
	var mindmapInfo, err = mm.MindmapStore.MindmapGet(name)
	if err != nil {
		return fmt.Errorf("failed to get mindmap info: %w", err)
	}

	// Check if the user has permission to access the mindmap
	if !mindmapInfo.IsPublic && mindmapInfo.Owner != mm.CurrentUser {
		return fmt.Errorf("user %s does not have permission to access mindmap '%s'", mm.CurrentUser, name)
	}

	// Load nodes for the selected mindmap
	nodes, err := mm.NodeStore.NodeGetAll(name, mm.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to load nodes for mindmap '%s': %w", name, err)
	}

	// Build the mindmap structure
	mindmap := &models.Mindmap{
		ID:       mindmapInfo.ID,
		Name:     mindmapInfo.Name,
		IsPublic: mindmapInfo.IsPublic,
		Owner:    mindmapInfo.Owner,
		Nodes:    make(map[int]*models.Node),
	}

	for _, node := range nodes {
		mindmap.Nodes[node.ID] = node
		if node.ParentID == -1 {
			mindmap.Root = node
		} else {
			parent := mindmap.Nodes[node.ParentID]
			if parent != nil {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	mm.CurrentMindmap = mindmap
	mm.Mindmaps[name] = mindmap
	mm.NodeManager.NodeReset()

	return nil
}

func (mm *MindmapManager) MindmapReset() {
	mm.Mindmaps = make(map[string]*models.Mindmap)
	mm.CurrentMindmap = nil
	mm.NodeManager.NodeReset()
}

func (mm *MindmapManager) MindmapList() ([]models.MindmapInfo, error) {
	if mm.CurrentUser == "" {
		return nil, fmt.Errorf("no user selected, please select a user first")
	}
	mindmaps, err := mm.MindmapStore.MindmapGetAll(mm.CurrentUser)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve mindmaps: %w", err)
	}

	// Filter out private mindmaps that don't belong to the current user
	var filteredMindmaps []models.MindmapInfo
	for _, m := range mindmaps {
		if m.IsPublic || m.Owner == mm.CurrentUser {
			filteredMindmaps = append(filteredMindmaps, m)
		}
	}

	return filteredMindmaps, nil
}

func (mm *MindmapManager) MindmapPermission(name string, setPublic *bool) (bool, error) {
	if setPublic == nil {
		// Just check permission without modifying
		hasPermission, err := mm.MindmapStore.MindmapPermission(name, mm.CurrentUser)
		if err != nil {
			return false, fmt.Errorf("failed to check mindmap permission: %w", err)
		}
		return hasPermission, nil
	}

	// Modify permission
	hasPermission, err := mm.MindmapStore.MindmapPermission(name, mm.CurrentUser, *setPublic)
	if err != nil {
		return false, fmt.Errorf("failed to update mindmap permission: %w", err)
	}
	if !hasPermission {
		return false, fmt.Errorf("you don't have permission to modify mindmap '%s'", name)
	}
	return true, nil
}

func (mm *MindmapManager) MindmapExport(filename, format string) error {
	if mm.CurrentMindmap == nil {
		return fmt.Errorf("no mindmap selected")
	}

	if !mm.CurrentMindmap.IsPublic && mm.CurrentMindmap.Owner != mm.CurrentUser {
		return fmt.Errorf("you don't have permission to export mindmap: %s", mm.CurrentMindmap.Name)
	}

	err := storage.FileExport(mm.MindmapStore, mm.NodeStore, mm.CurrentMindmap.Name, mm.CurrentUser, filename, format)
	if err != nil {
		return fmt.Errorf("failed to export mindmap: %w", err)
	}

	return nil
}

func (mm *MindmapManager) MindmapImport(filename, format string) error {
	// Deselect current mindmap
	err := mm.MindmapSelect("")
	if err != nil {
		return fmt.Errorf("failed to deselect current mindmap: %w", err)
	}

	// Import mindmap
	importedMindmap, err := storage.FileImport(mm.MindmapStore, mm.NodeStore, mm.CurrentUser, filename, format)
	if err != nil {
		return fmt.Errorf("failed to import mindmap: %w", err)
	}

	// Reset MaxID and reassign IDs
	maxID := 0
	var reassignIDs func(*models.Node)
	reassignIDs = func(node *models.Node) {
		node.ID = maxID
		maxID++
		for _, child := range node.Children {
			reassignIDs(child)
		}
	}
	reassignIDs(importedMindmap.Root)
	importedMindmap.MaxID = maxID - 1

	// Recreate the Nodes map
	importedMindmap.Nodes = make(map[int]*models.Node)
	var addToNodesMap func(*models.Node)
	addToNodesMap = func(node *models.Node) {
		importedMindmap.Nodes[node.ID] = node
		for _, child := range node.Children {
			addToNodesMap(child)
		}
	}
	addToNodesMap(importedMindmap.Root)

	// Add the imported mindmap to the in-memory map
	mm.Mindmaps[importedMindmap.Name] = importedMindmap

	// Persist the updated mindmap to storage
	err = mm.persistImportedMindmap(importedMindmap)
	if err != nil {
		return fmt.Errorf("failed to persist imported mindmap: %w", err)
	}

	return nil
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

func (mm *MindmapManager) persistImportedMindmap(mindmap *models.Mindmap) error {
	// Delete existing mindmap if it exists
	err := mm.MindmapStore.MindmapDelete(mindmap.Name, mm.CurrentUser)
	if err != nil && !strings.Contains(err.Error(), "not exist") {
		return fmt.Errorf("failed to delete existing mindmap: %w", err)
	}

	// Add new mindmap
	_, err = mm.MindmapStore.MindmapAdd(mindmap.Name, mm.CurrentUser, mindmap.IsPublic)
	if err != nil {
		return fmt.Errorf("failed to add imported mindmap: %w", err)
	}

	// Add nodes
	var addNode func(*models.Node) error
	addNode = func(node *models.Node) error {
		newID, err := mm.NodeStore.NodeAdd(mindmap.Name, mm.CurrentUser, node.ParentID, node.Content, node.Extra, node.Index)
		if err != nil {
			return fmt.Errorf("failed to add node: %w", err)
		}
		node.ID = newID
		for _, child := range node.Children {
			child.ParentID = newID
			err = addNode(child)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return addNode(mindmap.Root)
}
