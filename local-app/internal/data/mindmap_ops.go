// Package data provides data management functionality for the Mindnoscape application.
// This file contains operations related to mindmap management.
package data

import (
	"fmt"
	"mindnoscape/local-app/internal/models"
	"mindnoscape/local-app/internal/storage"
	"strconv"
)

// MindmapOperations defines the interface for mindmap-related operations
type MindmapOperations interface {
	MindmapAdd(name string, isPublic bool) error
	MindmapDelete(name string) error
	MindmapSelect(name string) error
	MindmapReset()
	MindmapGet(name string) *models.MindmapInfo
	MindmapList() ([]models.MindmapInfo, error)
	MindmapPermission(name string, username string, setPublic ...bool) (bool, error)
	MindmapExport(filename, format string) error
	MindmapImport(filename, format string) error
}

// MindmapManager handles all mindmap-related operations and maintains the current mindmap state.
type MindmapManager struct {
	mindmapStore storage.MindmapStore
	nodeStore    storage.NodeStore

	nodeManager *NodeManager

	Mindmaps       map[string]*models.Mindmap
	currentMindmap *models.Mindmap
	currentUser    string
}

// NewMindmapManager creates a new MindmapManager instance.
func NewMindmapManager(mindmapStore storage.MindmapStore, nodeStore storage.NodeStore, username string) (*MindmapManager, error) {
	mm := &MindmapManager{
		mindmapStore: mindmapStore,
		nodeStore:    nodeStore,
		Mindmaps:     make(map[string]*models.Mindmap),
		currentUser:  username,
	}

	mm.nodeManager = NewNodeManager(mm)

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

// MindmapAdd creates a new mindmap with the given name and visibility.
func (mm *MindmapManager) MindmapAdd(name string, isPublic bool) error {
	// Check if a user is selected
	if mm.currentUser == "" {
		return fmt.Errorf("no user selected, please select a user first")
	}

	// Check if the mindmap already exists
	if _, exists := mm.Mindmaps[name]; exists {
		return fmt.Errorf("mindmap '%s' already exists", name)
	}

	// Add to storage and get the new MindmapID
	mindmapID, err := mm.mindmapStore.MindmapAdd(name, mm.currentUser, isPublic)
	if err != nil {
		return fmt.Errorf("failed to add mindmap to storage: %w", err)
	}

	newMindmap := models.NewMindmap(mindmapID, name, mm.currentUser, isPublic)

	// Create root node
	root := models.NewNode(0, name, mindmapID)
	root.ParentID = -1
	root.Index = "0"

	rootID, err := mm.nodeStore.NodeAdd(name, mm.currentUser, -1, root.Content, root.Extra, root.Index)
	if err != nil {
		return fmt.Errorf("failed to add root node: %w", err)
	}
	root.ID = rootID
	newMindmap.Root = root
	newMindmap.Nodes[rootID] = root

	mm.Mindmaps[name] = newMindmap

	return nil
}

// MindmapDelete removes a mindmap and all its associated data.
func (mm *MindmapManager) MindmapDelete(name string) error {
	// Check if a user is selected
	if mm.currentUser == "" {
		return fmt.Errorf("no user selected, please select a user first")
	}

	// Check if the mindmap exists and the user has permission to delete it
	mindmapInfo, err := mm.mindmapStore.MindmapGet(name)
	if err != nil {
		return fmt.Errorf("failed to get mindmap info: %w", err)
	}

	if mindmapInfo.Owner != mm.currentUser {
		return fmt.Errorf("user '%s' does not have permission to delete mindmap '%s'", mm.currentUser, name)
	}

	// Delete the mindmap and all associated data
	err = mm.mindmapStore.MindmapDelete(name, mm.currentUser)
	if err != nil {
		return fmt.Errorf("failed to delete mindmap from storage: %w", err)
	}

	// Remove the mindmap from the in-memory map
	delete(mm.Mindmaps, name)

	// If the deleted mindmap was the current mindmap, reset the current mindmap
	if mm.currentMindmap != nil && mm.currentMindmap.Name == name {
		mm.currentMindmap = nil
		mm.nodeManager.NodeReset()
	}

	return nil
}

// MindmapSelect sets the current mindmap.
func (mm *MindmapManager) MindmapSelect(name string) error {
	// Check if a user is selected
	if mm.currentUser == "" {
		return fmt.Errorf("no user selected, please select a user first")
	}

	// If name is empty, reset the current mindmap
	if name == "" {
		mm.MindmapReset()
		return nil
	}

	// Load mindmap details
	var mindmapInfo, err = mm.mindmapStore.MindmapGet(name)
	if err != nil {
		return fmt.Errorf("failed to get mindmap info: %w", err)
	}

	// Check if the user has permission to access the mindmap
	if !mindmapInfo.IsPublic && mindmapInfo.Owner != mm.currentUser {
		return fmt.Errorf("user %s does not have permission to access mindmap '%s'", mm.currentUser, name)
	}

	// Load nodes for the selected mindmap
	nodes, err := mm.nodeStore.NodeGetAll(name, mm.currentUser)
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

	// Set the current mindmap and reset the node manager
	mm.currentMindmap = mindmap
	mm.Mindmaps[name] = mindmap
	mm.nodeManager.NodeReset()

	return nil
}

// MindmapReset clears the current mindmap selection and associated data.
func (mm *MindmapManager) MindmapReset() {
	mm.Mindmaps = make(map[string]*models.Mindmap)
	mm.currentMindmap = nil
	mm.nodeManager.NodeReset()
}

// MindmapGet retrieves the information of a specific mindmap.
func (mm *MindmapManager) MindmapGet() *models.MindmapInfo {
	if mm.currentMindmap == nil {
		return nil
	}
	return &models.MindmapInfo{
		ID:       mm.currentMindmap.ID,
		Name:     mm.currentMindmap.Name,
		IsPublic: mm.currentMindmap.IsPublic,
		Owner:    mm.currentMindmap.Owner,
	}
}

// MindmapList returns a list of all accessible mindmaps for the current user.
func (mm *MindmapManager) MindmapList() ([]models.MindmapInfo, error) {
	// Check if a user is selected
	if mm.currentUser == "" {
		return nil, fmt.Errorf("no user selected, please select a user first")
	}

	// Retrieve all mindmaps
	mindmaps, err := mm.mindmapStore.MindmapGetAll(mm.currentUser)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve mindmaps: %w", err)
	}

	// Filter out private mindmaps that don't belong to the current user
	var filteredMindmaps []models.MindmapInfo
	for _, m := range mindmaps {
		if m.IsPublic || m.Owner == mm.currentUser {
			filteredMindmaps = append(filteredMindmaps, m)
		}
	}

	return filteredMindmaps, nil
}

// MindmapPermission checks or sets the permission of a mindmap.
func (mm *MindmapManager) MindmapPermission(name string, setPublic *bool) (bool, error) {
	if setPublic == nil {
		// Just check permission without modifying
		hasPermission, err := mm.mindmapStore.MindmapPermission(name, mm.currentUser)
		if err != nil {
			return false, fmt.Errorf("failed to check mindmap permission: %w", err)
		}
		return hasPermission, nil
	}

	// Modify permission
	hasPermission, err := mm.mindmapStore.MindmapPermission(name, mm.currentUser, *setPublic)
	if err != nil {
		return false, fmt.Errorf("failed to update mindmap permission: %w", err)
	}
	if !hasPermission {
		return false, fmt.Errorf("you don't have permission to modify mindmap '%s'", name)
	}
	return true, nil
}

// MindmapExport exports the current mindmap to a file in the specified format.
func (mm *MindmapManager) MindmapExport(filename, format string) error {
	// Check if a user is selected
	if mm.currentMindmap == nil {
		return fmt.Errorf("no mindmap selected")
	}

	// Check if the user has permission to export the mindmap
	if !mm.currentMindmap.IsPublic && mm.currentMindmap.Owner != mm.currentUser {
		return fmt.Errorf("you don't have permission to export mindmap: %s", mm.currentMindmap.Name)
	}

	// Check if the user has permission to export the mindmap
	err := storage.FileExport(mm.mindmapStore, mm.nodeStore, mm.currentMindmap.Name, mm.currentUser, filename, format)
	if err != nil {
		return fmt.Errorf("failed to export mindmap: %w", err)
	}

	return nil
}

// MindmapImport imports a mindmap from a file in the specified format.
func (mm *MindmapManager) MindmapImport(filename, format string) (*models.MindmapInfo, error) {
	// Import mindmap
	importedMindmap, err := storage.FileImport(filename, format)
	if err != nil {
		return nil, fmt.Errorf("failed to import mindmap: %w", err)
	}

	// Validate the imported mindmap structure
	if err := mm.validateMindmap(importedMindmap); err != nil {
		return nil, fmt.Errorf("invalid mindmap structure: %w", err)
	}

	// Check if a mindmap with the same name exists
	exists, err := mm.mindmapStore.MindmapExists(importedMindmap.Name, mm.currentUser)
	if err != nil {
		return nil, fmt.Errorf("failed to check mindmap existence: %w", err)
	}
	if exists {
		// If it exists, delete it first
		err = mm.MindmapDelete(importedMindmap.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to delete existing mindmap: %w", err)
		}
	}

	// Create the mindmap (this will also create the root node)
	err = mm.MindmapAdd(importedMindmap.Root.Content, importedMindmap.IsPublic)
	if err != nil {
		return nil, fmt.Errorf("failed to create imported mindmap: %w", err)
	}

	// Select the newly created mindmap
	err = mm.MindmapSelect(importedMindmap.Root.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to select imported mindmap: %w", err)
	}

	// Update root node if necessary
	if len(importedMindmap.Root.Extra) > 0 {
		err = mm.nodeManager.NodeUpdate("0", importedMindmap.Root.Content, importedMindmap.Root.Extra, true, true)
		if err != nil {
			return nil, fmt.Errorf("failed to update root node: %w", err)
		}
	}

	// Add non-root nodes
	var addNode func(*models.Node) error
	addNode = func(node *models.Node) error {
		if node.ID == 0 {
			// Skip root node as it's already handled
			return nil
		}
		// Pass the original node ID to NodeAdd
		err := mm.nodeManager.NodeAdd(strconv.Itoa(node.ParentID), node.Content, node.Extra, true, true, node.ID)
		if err != nil {
			return fmt.Errorf("failed to add node %d: %w", node.ID, err)
		}
		for _, child := range node.Children {
			err = addNode(child)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Start adding nodes from the root's children
	for _, child := range importedMindmap.Root.Children {
		err = addNode(child)
		if err != nil {
			return nil, err
		}
	}

	// Get the updated mindmap info
	updatedMindmap := mm.MindmapGet()
	if updatedMindmap == nil {
		return nil, fmt.Errorf("failed to get updated mindmap info")
	}

	return updatedMindmap, nil
}

func (mm *MindmapManager) validateMindmap(mindmap *models.Mindmap) error {
	// Check root node
	if mindmap.Root.ID != 0 || mindmap.Root.ParentID != -1 || mindmap.Root.Index != "0" {
		return fmt.Errorf("invalid root node structure")
	}

	nodeIDs := make(map[int]bool)
	nodeIDs[0] = true // Root node

	// First pass: Check for duplicate IDs
	var checkDuplicateIDs func(*models.Node) error
	checkDuplicateIDs = func(node *models.Node) error {
		if node.ID != 0 {
			if nodeIDs[node.ID] {
				return fmt.Errorf("duplicate node ID: %d", node.ID)
			}
			nodeIDs[node.ID] = true
		}
		for _, child := range node.Children {
			if err := checkDuplicateIDs(child); err != nil {
				return err
			}
		}
		return nil
	}

	if err := checkDuplicateIDs(mindmap.Root); err != nil {
		return err
	}

	// Second pass: Validate parent IDs
	var validateParentIDs func(*models.Node) error
	validateParentIDs = func(node *models.Node) error {
		if node.ID != 0 {
			if !nodeIDs[node.ParentID] {
				return fmt.Errorf("invalid parent ID for node %d: parent %d not found", node.ID, node.ParentID)
			}
		}
		for _, child := range node.Children {
			if err := validateParentIDs(child); err != nil {
				return err
			}
		}
		return nil
	}

	return validateParentIDs(mindmap.Root)
}
