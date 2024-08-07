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
		data, err = xml.MarshalIndent(xmlRoot, "", "  ")
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
		err = xml.Unmarshal(data, &xmlRoot)
		if err == nil {
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
			root = convertFromXMLNode(xmlRoot)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal mind map: %v", err)
	}

	return root, nil
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
		if node.Index == 0 {
			root = node
		}
	}

	for _, node := range nodes {
		if node.Index != 0 { // Skip root
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
	// First, import the file into a temporary root node
	tempRoot, err := ImportFromFile(filename, format)
	if err != nil {
		return err
	}

	// Clear existing data
	if err := store.ClearAllNodes(); err != nil {
		return fmt.Errorf("failed to clear existing nodes: %v", err)
	}

	// Insert new data
	if err := insertNodeRecursive(store, tempRoot, -1); err != nil {
		return err
	}

	return nil
}

func insertNodeRecursive(store Store, node *models.Node, parentID int) error {
	var err error
	if parentID == -1 {
		// This is the root node
		err = store.UpdateNode(0, node.Content, node.Extra, node.LogicalIndex)
	} else {
		err = store.AddNode(parentID, node.Content, node.Extra, node.LogicalIndex)
	}

	if err != nil {
		return fmt.Errorf("failed to add/update node: %v", err)
	}

	fmt.Printf("Inserted/Updated node: Content=%s, LogicalIndex=%s\n", node.Content, node.LogicalIndex)

	for _, child := range node.Children {
		err = insertNodeRecursive(store, child, node.Index)
		if err != nil {
			return err
		}
	}

	return nil
}
