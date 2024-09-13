package session

import (
	"errors"
	"fmt"
	"mindnoscape/local-app/src/pkg/event"
	"mindnoscape/local-app/src/pkg/model"
	"strings"
)

// handleMindmapAdd handles the mindmap add command
func handleMindmapAdd(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) != 1 {
		return nil, errors.New("mindmap add command requires exactly 1 argument: <mindmap_name>")
	}

	user, err := s.UserGet()
	if err != nil {
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	mindmapInfo := model.MindmapInfo{
		Name: cmd.Args[0],
	}

	mindmapID, err := s.DataManager.MindmapManager.MindmapAdd(user, mindmapInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to add mindmap: %w", err)
	}

	// Fetch the newly created mindmap and set it as the current mindmap
	mindmaps, err := s.DataManager.MindmapManager.MindmapGet(user, model.MindmapInfo{ID: mindmapID}, model.MindmapFilter{ID: true})
	if err != nil || len(mindmaps) == 0 {
		return nil, fmt.Errorf("failed to retrieve newly created mindmap: %w", err)
	}
	s.MindmapSet(mindmaps[0])

	return mindmapID, nil
}

// handleMindmapDelete handles the mindmap delete command
func handleMindmapDelete(s *Session, cmd model.Command) (interface{}, error) {
	user, err := s.UserGet()
	if err != nil {
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	if len(cmd.Args) == 0 {
		// Delete current mindmap
		currentMindmap, err := s.MindmapGet()
		if err != nil {
			return nil, fmt.Errorf("no mindmap selected: %w", err)
		}
		err = s.DataManager.MindmapManager.MindmapDelete(user, currentMindmap)
		if err != nil {
			return nil, fmt.Errorf("failed to delete current mindmap: %w", err)
		}
		s.MindmapSet(nil)
		return nil, nil
	}

	// Delete specific mindmap
	mindmapName := cmd.Args[0]
	mindmaps, err := s.DataManager.MindmapManager.MindmapGet(user, model.MindmapInfo{Name: mindmapName}, model.MindmapFilter{Name: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get mindmap: %w", err)
	}
	if len(mindmaps) == 0 {
		return nil, fmt.Errorf("mindmap not found: %s", mindmapName)
	}

	err = s.DataManager.MindmapManager.MindmapDelete(user, mindmaps[0])
	if err != nil {
		return nil, fmt.Errorf("failed to delete mindmap: %w", err)
	}

	// If the deleted mindmap was the current one, clear it from the session
	currentMindmap, _ := s.MindmapGet()
	if currentMindmap != nil && currentMindmap.ID == mindmaps[0].ID {
		s.MindmapSet(nil)
	}

	return nil, nil
}

// handleMindmapPermission handles the mindmap permission command
func handleMindmapPermission(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		return nil, errors.New("mindmap permission command requires 1 or 2 arguments: <mindmap_name> [public|private]")
	}

	user, err := s.UserGet()
	if err != nil {
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	mindmapName := cmd.Args[0]
	mindmaps, err := s.DataManager.MindmapManager.MindmapGet(user, model.MindmapInfo{Name: mindmapName}, model.MindmapFilter{Name: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get mindmap: %w", err)
	}
	if len(mindmaps) == 0 {
		return nil, fmt.Errorf("mindmap not found: %s", mindmapName)
	}
	mindmap := mindmaps[0]

	if len(cmd.Args) == 1 {
		// Return current permission
		return mindmap.IsPublic, nil
	}

	// Set new permission
	isPublic := strings.ToLower(cmd.Args[1]) == "public"
	err = s.DataManager.MindmapManager.MindmapUpdate(user, mindmap, model.MindmapInfo{IsPublic: isPublic}, model.MindmapFilter{IsPublic: true})
	if err != nil {
		return nil, fmt.Errorf("failed to update mindmap permission: %w", err)
	}

	// Update the session's Mindmap object if it's the current mindmap
	currentMindmap, _ := s.MindmapGet()
	if currentMindmap != nil && currentMindmap.ID == mindmap.ID {
		currentMindmap.IsPublic = isPublic
	}

	return isPublic, nil
}

// handleMindmapImport handles the mindmap import command
func handleMindmapImport(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		return nil, errors.New("mindmap import command requires 1 or 2 arguments: <filename> [json|xml]")
	}

	user, err := s.UserGet()
	if err != nil {
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	filename := cmd.Args[0]
	format := "json"
	if len(cmd.Args) == 2 {
		format = strings.ToLower(cmd.Args[1])
	}

	if format != "json" && format != "xml" {
		return nil, fmt.Errorf("invalid format: %s. Must be 'json' or 'xml'", format)
	}

	importedMindmap, err := s.DataManager.MindmapImport(user, filename, format)
	if err != nil {
		return nil, fmt.Errorf("failed to import mindmap: %w", err)
	}

	// Set the imported mindmap as the current mindmap
	s.MindmapSet(importedMindmap)

	return importedMindmap, nil
}

// handleMindmapExport handles the mindmap export command
func handleMindmapExport(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		return nil, errors.New("mindmap export command requires 1 or 2 arguments: <filename> [json|xml]")
	}

	user, err := s.UserGet()
	if err != nil {
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	currentMindmap, err := s.MindmapGet()
	if err != nil {
		return nil, fmt.Errorf("no mindmap selected: %w", err)
	}

	filename := cmd.Args[0]
	format := "json"
	if len(cmd.Args) == 2 {
		format = strings.ToLower(cmd.Args[1])
	}

	if format != "json" && format != "xml" {
		return nil, fmt.Errorf("invalid format: %s. Must be 'json' or 'xml'", format)
	}

	err = s.DataManager.MindmapExport(user, currentMindmap, filename, format)
	if err != nil {
		return nil, fmt.Errorf("failed to export mindmap: %w", err)
	}

	return nil, nil
}

// handleMindmapSelect handles the mindmap select command
func handleMindmapSelect(s *Session, cmd model.Command) (interface{}, error) {
	user, err := s.UserGet()
	if err != nil {
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	if len(cmd.Args) == 0 {
		// Deselect current mindmap
		s.MindmapSet(nil)
		return nil, nil
	}

	mindmapName := cmd.Args[0]
	mindmaps, err := s.DataManager.MindmapManager.MindmapGet(user, model.MindmapInfo{Name: mindmapName}, model.MindmapFilter{Name: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get mindmap: %w", err)
	}
	if len(mindmaps) == 0 {
		return nil, fmt.Errorf("mindmap not found: %s", mindmapName)
	}

	selectedMindmap := mindmaps[0]
	s.MindmapSet(selectedMindmap)

	// Publish MindmapSelected event
	s.DataManager.EventManager.Publish(event.Event{
		Type: event.MindmapSelected,
		Data: selectedMindmap,
	})

	return mindmaps[0], nil
}

// handleMindmapList handles the mindmap list command
func handleMindmapList(s *Session, cmd model.Command) (interface{}, error) {
	user, err := s.UserGet()
	if err != nil {
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	mindmaps, err := s.DataManager.MindmapManager.MindmapGet(user, model.MindmapInfo{}, model.MindmapFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to get mindmaps: %w", err)
	}

	return mindmaps, nil
}

// handleMindmapView handles the mindmap view command
func handleMindmapView(s *Session, cmd model.Command) (interface{}, error) {
	currentMindmap, err := s.MindmapGet()
	if err != nil {
		return nil, fmt.Errorf("no mindmap selected: %w", err)
	}

	showID := false
	var node *model.Node

	for _, arg := range cmd.Args {
		if arg == "--id" {
			showID = true
		} else {
			// Assume the argument is an index
			nodes, err := s.DataManager.NodeManager.NodeGet(currentMindmap, model.NodeInfo{Index: arg}, model.NodeFilter{Index: true})
			if err != nil {
				return nil, fmt.Errorf("failed to get node: %w", err)
			}
			if len(nodes) == 0 {
				return nil, fmt.Errorf("node not found with index: %s", arg)
			}
			node = nodes[0]
		}
	}

	if node == nil {
		node = currentMindmap.Root
	}

	// Here you would implement the logic to format the node and its children for display
	// This is a placeholder implementation
	return formatNodeForDisplay(node, showID), nil
}

// formatNodeForDisplay is a helper function to format a node and its children for display
// This is a placeholder implementation and should be replaced with actual formatting logic
func formatNodeForDisplay(node *model.Node, showID bool) string {
	// Implement the logic to format the node and its children for display
	// This could involve recursively traversing the node's children and creating a string representation
	return fmt.Sprintf("Node: %s (ID: %d)", node.Name, node.ID)
}
