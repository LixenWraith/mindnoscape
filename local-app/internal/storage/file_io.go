package storage

import (
	"fmt"
	"os"

	"mindnoscape/local-app/internal/models"

	"encoding/json"
	"encoding/xml"
)

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

	root, err := buildTreeFromNodes(nodes)
	if err != nil {
		return fmt.Errorf("failed to build tree: %w", err)
	}

	mindmap := &models.Mindmap{
		Name: mindmapName,
		Root: root,
	}

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

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func FileImport(mindmapStore MindmapStore, nodeStore NodeStore, username, filename, format string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var root *models.Node
	switch format {
	case "json":
		var mindmap models.Mindmap
		err = json.Unmarshal(data, &mindmap)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
		root = mindmap.Root
	case "xml":
		root, err = unmarshalFromXML(data)
		if err != nil {
			return fmt.Errorf("failed to unmarshal XML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	mindmapName := root.Content

	// Check if the mindmap exists
	exists, err := mindmapStore.MindmapExists(mindmapName, username)
	if err != nil {
		return fmt.Errorf("failed to check if mindmap exists: %w", err)
	}

	// If the mindmap doesn't exist, create it
	if !exists {
		_, err = mindmapStore.MindmapAdd(mindmapName, username, false) // Use username and set isPublic to false
		if err != nil {
			return fmt.Errorf("failed to create mindmap: %w", err)
		}
	} else {
		// If it exists, delete all existing nodes
		nodes, err := nodeStore.NodeGetAll(mindmapName, username)
		if err != nil {
			return fmt.Errorf("failed to get existing nodes: %w", err)
		}
		for _, node := range nodes {
			err = nodeStore.NodeDelete(mindmapName, username, node.Index)
			if err != nil {
				return fmt.Errorf("failed to delete existing node: %w", err)
			}
		}
	}

	// Insert new nodes
	err = insertNodeRecursive(nodeStore, mindmapName, username, root, -1)
	if err != nil {
		return fmt.Errorf("failed to insert nodes for mindmap '%s': %w", mindmapName, err)
	}

	return nil
}

func buildTreeFromNodes(nodes []*models.Node) (*models.Node, error) {
	nodeMap := make(map[int]*models.Node)
	var root *models.Node

	for _, node := range nodes {
		nodeMap[node.Index] = node
		if node.Index == 0 || node.ParentID == -1 || node.LogicalIndex == "0" {
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
				return nil, fmt.Errorf("parent node not found for node %d (content: %s, logical index: %s)", node.Index, node.Content, node.LogicalIndex)
			}
			parent.Children = append(parent.Children, node)
		}
	}

	return root, nil
}

func insertNodeRecursive(nodeStore NodeStore, mindmapName string, username string, node *models.Node, parentID int) error {
	err := nodeStore.NodeAdd(mindmapName, username, parentID, node.Content, node.Extra, node.LogicalIndex)
	if err != nil {
		return err
	}

	for _, child := range node.Children {
		err = insertNodeRecursive(nodeStore, mindmapName, username, child, node.Index)
		if err != nil {
			return err
		}
	}

	return nil
}

// marshalToXML converts a Node structure to XML format.
// It recursively processes the node and its children, including extra fields.
func marshalToXML(root *models.Node) ([]byte, error) {
	type xmlField struct {
		Key   string `xml:"key,attr"`
		Value string `xml:",chardata"`
	}
	type xmlNode struct {
		Index        int        `xml:"index,attr"`
		ParentID     int        `xml:"parent_id,attr"`
		Content      string     `xml:"content"`
		LogicalIndex string     `xml:"logical_index,attr"`
		Children     []xmlNode  `xml:"children>node,omitempty"`
		Extra        []xmlField `xml:"extra>field,omitempty"`
	}

	var convertToXMLNode func(*models.Node) xmlNode
	convertToXMLNode = func(n *models.Node) xmlNode {
		xn := xmlNode{
			Index:        n.Index,
			ParentID:     n.ParentID,
			Content:      n.Content,
			LogicalIndex: n.LogicalIndex,
		}
		for k, v := range n.Extra {
			xn.Extra = append(xn.Extra, xmlField{Key: k, Value: v})
		}
		for _, child := range n.Children {
			xn.Children = append(xn.Children, convertToXMLNode(child))
		}
		return xn
	}

	xmlRoot := convertToXMLNode(root)
	return xml.MarshalIndent(xmlRoot, "", "  ")
}

// unmarshalFromXML converts XML data back into a Node structure.
// It recursively rebuilds the node hierarchy and extra fields.
func unmarshalFromXML(data []byte) (*models.Node, error) {
	type xmlField struct {
		Key   string `xml:"key,attr"`
		Value string `xml:",chardata"`
	}
	type xmlNode struct {
		Index        int        `xml:"index,attr"`
		ParentID     int        `xml:"parent_id,attr"`
		Content      string     `xml:"content"`
		LogicalIndex string     `xml:"logical_index,attr"`
		Children     []xmlNode  `xml:"children>node,omitempty"`
		Extra        []xmlField `xml:"extra>field,omitempty"`
	}

	var xmlRoot xmlNode
	err := xml.Unmarshal(data, &xmlRoot)
	if err != nil {
		return nil, err
	}

	var convertFromXMLNode func(xmlNode) *models.Node
	convertFromXMLNode = func(xn xmlNode) *models.Node {
		n := &models.Node{
			Index:        xn.Index,
			ParentID:     xn.ParentID,
			Content:      xn.Content,
			LogicalIndex: xn.LogicalIndex,
			Extra:        make(map[string]string),
		}
		for _, field := range xn.Extra {
			n.Extra[field.Key] = field.Value
		}
		for _, child := range xn.Children {
			n.Children = append(n.Children, convertFromXMLNode(child))
		}
		return n
	}

	return convertFromXMLNode(xmlRoot), nil
}
