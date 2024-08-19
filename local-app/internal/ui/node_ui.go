// Package ui provides user interface functionality for the Mindnoscape application.
// This file contains the NodeUI struct and methods for displaying node-related information.
package ui

import (
	"fmt"
	"io"
	"strings"

	"mindnoscape/local-app/internal/models"
)

// NodeUI handles the visualization of node-related information.
type NodeUI struct {
	visualizer *Visualizer
}

// NewNodeUI creates a new NodeUI instance.
func NewNodeUI(w io.Writer, useColor bool) *NodeUI {
	return &NodeUI{
		visualizer: NewVisualizer(w, useColor),
	}
}

// NodeInfo displays information about a single node
func (nui *NodeUI) NodeInfo(node *models.Node) {
	nui.visualizer.Printf("Node Content: %s\n", node.Content)
	nui.visualizer.Printf("Node Index: %d\n", node.Index)
	nui.visualizer.Printf("Logical Index: %s\n", node.Index)
	if len(node.Extra) > 0 {
		nui.visualizer.Println("Extra fields:")
		for key, value := range node.Extra {
			nui.visualizer.Printf("  %s: %s\n", key, value)
		}
	}
}

// NodeFind displays the results of a node search
func (nui *NodeUI) NodeFind(matches []*models.Node, showIndex bool) {
	if len(matches) == 0 {
		nui.visualizer.Println("No matches found.")
		return
	}

	nui.visualizer.Printf("Found %d matches:\n", len(matches))
	for _, node := range matches {
		nui.displayNodeLine(node, showIndex)
	}
}

// displayNodeLine displays a single line of node information.
func (nui *NodeUI) displayNodeLine(node *models.Node, showID bool) {
	// Construct the base line with node id (if required), index, and content
	line := fmt.Sprintf("{{yellow}}%s{{default}} %s %s", node.Index, func(show bool) string {
		idPart := fmt.Sprintf("{{orange}}[%d]{{default}}", node.ID)
		if show == true {
			return idPart
		} else {
			return ""
		}
	}(showID), node.Content)

	// Add extra fields if any
	if len(node.Extra) > 0 {
		var extraFields []string
		for k, v := range node.Extra {
			extraFields = append(extraFields, fmt.Sprintf("%s:%s", k, v))
		}
		line += " " + strings.Join(extraFields, ", ")
	}

	// Display the constructed line
	nui.visualizer.PrintMultiColoredLine(line, nui.getColorMap())
}

// getColorMap returns a map of color codes used in node visualization.
func (nui *NodeUI) getColorMap() map[string]Color {
	return map[string]Color{
		"{{yellow}}":  ColorYellow,
		"{{orange}}":  ColorOrange,
		"{{default}}": ColorDefault,
	}
}