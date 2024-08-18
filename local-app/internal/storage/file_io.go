// Package storage provides functionality for persisting and retrieving Mindnoscape data.
// This file handles the import and export of mindmaps to and from files.
package storage

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"

	"mindnoscape/local-app/internal/models"
)

// FileExport exports a mindmap to a file in the specified format (JSON or XML).
func FileExport(mindmapStore MindmapStore, nodeStore NodeStore, mindmapName, username, filename, format string) error {
	// Check if the mindmap exists and the user has permission
	exists, err := mindmapStore.MindmapExists(mindmapName, username)
	if err != nil {
		return fmt.Errorf("failed to check if mindmap exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("mindmap '%s' does not exist or user doesn't have permission", mindmapName)
	}

	// Get all nodes for the mindmap
	nodes, err := nodeStore.NodeGetAll(mindmapName, username)
	if err != nil {
		return fmt.Errorf("failed to get all nodes for mindmap '%s': %w", mindmapName, err)
	}

	// Build the tree structure from the nodes
	root, err := buildTreeFromNodes(nodes)
	if err != nil {
		return fmt.Errorf("failed to build tree: %w", err)
	}

	// Create the mindmap structure
	mindmap := &models.Mindmap{
		Name: mindmapName,
		Root: root,
	}

	// Marshal the mindmap to the specified format
	var data []byte
	switch format {
	case "json":
		data, err = json.MarshalIndent(mindmap, "", "  ")
	case "xml":
		data, err = xml.MarshalIndent(mindmap, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal mindmap: %w", err)
	}

	// Write the data to the file
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// FileImport imports a mindmap from a file in the specified format (JSON or XML).
func FileImport(filename, format string) (*models.Mindmap, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal the data into a mindmap structure
	var importedMindmap *models.Mindmap
	switch format {
	case "json":
		err = json.Unmarshal(data, &importedMindmap)
	case "xml":
		err = xml.Unmarshal(data, &importedMindmap)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return importedMindmap, nil
}

// buildTreeFromNodes constructs a tree structure from a flat list of nodes.
func buildTreeFromNodes(nodes []*models.Node) (*models.Node, error) {
	nodeMap := make(map[int]*models.Node)
	var root *models.Node

	for _, node := range nodes {
		nodeMap[node.ID] = node
		if node.ID == 0 || node.ParentID == -1 || node.Index == "0" {
			root = node
		}
	}

	if root == nil {
		return nil, fmt.Errorf("root node not found")
	}

	for _, node := range nodes {
		if node != root {
			parent, exists := nodeMap[node.ParentID]
			if !exists {
				return nil, fmt.Errorf("parent node not found for node %d (content: %s, index: %s)", node.Index, node.Content, node.Index)
			}
			parent.Children = append(parent.Children, node)
		}
	}

	return root, nil
}
