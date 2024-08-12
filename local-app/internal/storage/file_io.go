package storage

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"

	"mindnoscape/local-app/internal/models"
)

func ExportToFile(filename string, format string, mindmap *models.Mindmap) error {
	exportableMindmap := mindmap.ToExportable()
	var data []byte
	var err error

	switch format {
	case "json":
		data, err = json.MarshalIndent(exportableMindmap, "", "  ")
	case "xml":
		data, err = xml.MarshalIndent(exportableMindmap, "", "  ")
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

	var root *models.Node
	switch format {
	case "json":
		err = json.Unmarshal(data, &root)
	case "xml":
		root, err = unmarshalFromXML(data)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal mind map: %v", err)
	}

	return root, nil
}

func buildTreeFromNodes(nodes []*models.Node) (*models.Node, error) {
	nodeMap := make(map[int]*models.Node)
	var root *models.Node

	for _, node := range nodes {
		nodeMap[node.Index] = node
		if node.Index == 0 || node.LogicalIndex == "0" {
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

func SaveToFile(store Store, mindmapName string, username string, filename string, format string) error {
	nodes, err := store.GetAllNodesForMindmap(mindmapName, username)
	if err != nil {
		return fmt.Errorf("failed to get all nodes for mindmap '%s': %v", mindmapName, err)
	}

	root, err := buildTreeFromNodes(nodes)
	if err != nil {
		return fmt.Errorf("failed to build tree: %v", err)
	}

	mindmap := &models.Mindmap{
		Name: mindmapName,
		Root: root,
	}

	return ExportToFile(filename, format, mindmap)
}

func LoadFromFile(store Store, mindmapName string, username string, filename string, format string) error {
	// First, import the file into a temporary root node
	var root, err = ImportFromFile(filename, format)
	if err != nil {
		return err
	}

	// Check if the mindmap exists
	exists, err := store.MindmapExists(mindmapName, username)
	if err != nil {
		return fmt.Errorf("failed to check if mindmap exists: %v", err)
	}

	// If the mindmap doesn't exist, create it
	if !exists {
		_, err = store.AddMindmap(mindmapName, username, false) // Use username and set isPublic to false
		if err != nil {
			return fmt.Errorf("failed to create mindmap: %v", err)
		}
	}

	// Insert new data
	if err := insertNodeRecursive(store, mindmapName, root, -1); err != nil {
		return fmt.Errorf("failed to insert nodes for mindmap '%s': %v", mindmapName, err)
	}

	return nil
}

func insertNodeRecursive(store Store, mindmapName string, node *models.Node, parentID int) error {
	// Use an empty string for username as this is used during import
	err := store.AddNode(mindmapName, "", parentID, node.Content, node.Extra, node.LogicalIndex)
	if err != nil {
		return err
	}

	for _, child := range node.Children {
		err = insertNodeRecursive(store, mindmapName, child, node.Index)
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
