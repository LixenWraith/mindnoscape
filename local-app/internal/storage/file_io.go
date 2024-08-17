package storage

import (
	"fmt"
	"mindnoscape/local-app/internal/models"
	"os"

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

func FileImport(mindmapStore MindmapStore, nodeStore NodeStore, username, filename, format string) (*models.Mindmap, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

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

	// Check if a mindmap with the same name exists and delete it
	_, err = mindmapStore.MindmapGet(importedMindmap.Name)
	if err == nil {
		// Mindmap exists, delete it
		err = mindmapStore.MindmapDelete(importedMindmap.Name, username)
		if err != nil {
			return nil, fmt.Errorf("failed to delete existing mindmap: %w", err)
		}
	}

	// Create new mindmap
	mindmapID, err := mindmapStore.MindmapAdd(importedMindmap.Name, username, false) // Set as private
	if err != nil {
		return nil, fmt.Errorf("failed to create mindmap: %w", err)
	}

	// Insert nodes
	var insertNode func(*models.Node, int) error
	insertNode = func(node *models.Node, parentID int) error {
		newID, err := nodeStore.NodeAdd(importedMindmap.Name, username, parentID, node.Content, node.Extra, node.Index)
		if err != nil {
			return fmt.Errorf("failed to insert node: %w", err)
		}
		node.ID = newID
		for _, child := range node.Children {
			if err := insertNode(child, newID); err != nil {
				return err
			}
		}
		return nil
	}

	if err := insertNode(importedMindmap.Root, -1); err != nil {
		return nil, err
	}

	importedMindmap.Owner = username
	importedMindmap.IsPublic = false
	importedMindmap.ID = mindmapID

	return importedMindmap, nil
}

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

// marshalToXML converts a Node structure to XML format.
// It recursively processes the node and its children, including extra fields.
func marshalToXML(root *models.Node) ([]byte, error) {
	type xmlField struct {
		Key   string `xml:"key,attr"`
		Value string `xml:",chardata"`
	}
	type xmlNode struct {
		ID       int        `xml:"id,attr"`
		ParentID int        `xml:"parent_id,attr"`
		Content  string     `xml:"content"`
		Index    string     `xml:"index,attr"`
		Children []xmlNode  `xml:"children>node,omitempty"`
		Extra    []xmlField `xml:"extra>field,omitempty"`
	}

	var convertToXMLNode func(*models.Node) xmlNode
	convertToXMLNode = func(n *models.Node) xmlNode {
		xn := xmlNode{
			ID:       n.ID,
			ParentID: n.ParentID,
			Content:  n.Content,
			Index:    n.Index,
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
		ID       int        `xml:"id,attr"`
		ParentID int        `xml:"parent_id,attr"`
		Content  string     `xml:"content"`
		Index    string     `xml:"index,attr"`
		Children []xmlNode  `xml:"children>node,omitempty"`
		Extra    []xmlField `xml:"extra>field,omitempty"`
	}

	var xmlRoot xmlNode
	err := xml.Unmarshal(data, &xmlRoot)
	if err != nil {
		return nil, err
	}

	var convertFromXMLNode func(xmlNode) *models.Node
	convertFromXMLNode = func(xn xmlNode) *models.Node {
		n := &models.Node{
			ID:       xn.ID,
			ParentID: xn.ParentID,
			Content:  xn.Content,
			Index:    xn.Index,
			Extra:    make(map[string]string),
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
