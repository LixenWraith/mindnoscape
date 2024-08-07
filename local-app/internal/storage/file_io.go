package storage

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"

	"mindnoscape/local-app/internal/models"
)

func ExportToFile(root *models.Node, filename string, format string) error {
	var data []byte
	var err error

	switch format {
	case "json":
		data, err = json.MarshalIndent(root, "", "  ")
	case "xml":
		data, err = xml.MarshalIndent(root, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal mind map: %v", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

func ImportFromFile(filename string, format string) (*models.Node, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var root models.Node

	switch format {
	case "json":
		err = json.Unmarshal(data, &root)
	case "xml":
		err = xml.Unmarshal(data, &root)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal mind map: %v", err)
	}

	return &root, nil
}

func SaveToFile(store Store, filename string, format string) error {
	nodes, err := store.GetAllNodes()
	if err != nil {
		return fmt.Errorf("failed to get all nodes: %v", err)
	}

	// Reconstruct the tree structure
	nodeMap := make(map[int]*models.Node)
	var root *models.Node

	for _, node := range nodes {
		nodeMap[node.Index] = node
		if node.Index == 1 {
			root = node
		}
	}

	for _, node := range nodes {
		if node.Index != 1 { // Skip root
			parent := nodeMap[node.ParentID]
			if parent != nil {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	// Export the reconstructed tree
	return ExportToFile(root, filename, format)
}

func LoadFromFile(store Store, filename string, format string) error {
	root, err := ImportFromFile(filename, format)
	if err != nil {
		return err
	}

	// Clear existing data
	if err := store.ClearAllNodes(); err != nil {
		return fmt.Errorf("failed to clear existing nodes: %v", err)
	}

	// Insert new data
	return insertNodeRecursive(store, root, 0)
}

func insertNodeRecursive(store Store, node *models.Node, parentID int) error {
	// Generate a temporary logical index
	tempLogicalIndex := fmt.Sprintf("%d", node.Index)

	err := store.AddNode(parentID, node.Content, node.Extra, tempLogicalIndex)
	if err != nil {
		return fmt.Errorf("failed to add node: %v", err)
	}

	fmt.Printf("Inserted node: Content=%s, ParentID=%d\n", node.Content, parentID)

	for _, child := range node.Children {
		err = insertNodeRecursive(store, child, node.Index)
		if err != nil {
			return err
		}
	}

	return nil
}
