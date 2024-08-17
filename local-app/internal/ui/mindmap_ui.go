package ui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"mindnoscape/local-app/internal/models"
)

type MindmapUI struct {
	visualizer *Visualizer
}

func NewMindmapUI(w io.Writer, useColor bool) *MindmapUI {
	return &MindmapUI{
		visualizer: NewVisualizer(w, useColor),
	}
}

// MindmapList displays a list of mindmaps
func (mui *MindmapUI) MindmapList(mindmaps []models.MindmapInfo, currentUser string) {
	if len(mindmaps) == 0 {
		mui.visualizer.Println("No mindmaps available")
		return
	}

	mui.visualizer.Println("Available mindmaps:")
	for _, mm := range mindmaps {
		permissionSymbol := "+"
		permissionColor := ColorGreen
		if !mm.IsPublic {
			permissionSymbol = "-"
			permissionColor = ColorRed
		}

		mui.visualizer.Print(mm.Name + " ")
		mui.visualizer.PrintColored(permissionSymbol, permissionColor)
		if mm.Owner != currentUser {
			mui.visualizer.Printf(" (owner: %s)", mm.Owner)
		}
		mui.visualizer.Println("")
	}
}

// MindmapView displays the structure of a mindmap
func (mui *MindmapUI) MindmapView(nodes []*models.Node, showID bool) {
	if len(nodes) == 0 {
		mui.visualizer.Println("No nodes to display")
		return
	}

	// Start generating visualization from root
	output := mui.visualizeMindmap(nodes, showID)
	for _, line := range output {
		mui.visualizer.PrintMultiColoredLine(line, mui.getColorMap())
	}
}

func (mui *MindmapUI) visualizeMindmap(nodes []*models.Node, showID bool) []string {
	var output []string
	nodeMap := make(map[int]*models.Node)
	childrenMap := make(map[int][]*models.Node)

	// Create node and children maps
	for _, node := range nodes {
		nodeMap[node.ID] = node
		if node.ParentID != -1 {
			childrenMap[node.ParentID] = append(childrenMap[node.ParentID], node)
		}
	}

	// Find the root node
	var root *models.Node
	for _, node := range nodes {
		if node.ParentID == -1 {
			root = node
			break
		}
	}

	if root == nil {
		mui.visualizer.Println("Error: Root node not found")
		return output
	}

	// Helper function to build the tree
	var buildTree func(*models.Node, string, bool)
	buildTree = func(node *models.Node, prefix string, isLast bool) {
		var line strings.Builder
		line.WriteString(prefix)

		if isLast {
			line.WriteString("{{brown}}└── {{default}}")
			prefix += "    "
		} else {
			line.WriteString("{{brown}}├── {{default}}")
			prefix += "{{brown}}│   {{default}}"
		}

		line.WriteString(fmt.Sprintf("{{yellow}}%s{{default}}", node.Index))
		line.WriteString(" " + node.Content)

		if showID {
			line.WriteString(fmt.Sprintf(" {{orange}}[%d]{{default}}", node.ID))
		}

		if len(node.Extra) > 0 {
			var extraFields []string
			for k, v := range node.Extra {
				extraFields = append(extraFields, fmt.Sprintf("%s:%s", k, v))
			}
			line.WriteString(" " + strings.Join(extraFields, ", "))
		}

		output = append(output, line.String())

		children := childrenMap[node.ID]
		sort.Slice(children, func(i, j int) bool {
			return children[i].Index < children[j].Index
		})

		for i, child := range children {
			buildTree(child, prefix, i == len(children)-1)
		}
	}

	// Start building the tree from the root
	buildTree(root, "", true)

	return output
}

func (mui *MindmapUI) getColorMap() map[string]Color {
	return map[string]Color{
		"{{yellow}}":  ColorYellow,
		"{{orange}}":  ColorOrange,
		"{{brown}}":   ColorBrown,
		"{{default}}": ColorDefault,
	}
}
