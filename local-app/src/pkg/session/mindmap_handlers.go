package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"mindnoscape/local-app/src/pkg/event"
	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
)

// handleMindmapAdd handles the mindmap add command
func handleMindmapAdd(s *Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Handling mindmap add command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) != 1 {
		s.logger.Error(ctx, "Invalid number of arguments for mindmap add", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("mindmap add command requires exactly 1 argument: <mindmap_name>")
	}

	user, err := s.UserGet()
	if err != nil {
		s.logger.Error(ctx, "No user selected", log.Fields{"error": err})
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	mindmapInfo := model.MindmapInfo{
		Name: cmd.Args[0],
	}

	s.logger.Debug(ctx, "Adding new mindmap", log.Fields{"mindmapName": mindmapInfo.Name})
	mindmapID, err := s.DataManager.MindmapManager.MindmapAdd(user, mindmapInfo)
	if err != nil {
		s.logger.Error(ctx, "Failed to add mindmap", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to add mindmap: %w", err)
	}

	// Fetch the newly created mindmap and set it as the current mindmap
	mindmaps, err := s.DataManager.MindmapManager.MindmapGet(user, model.MindmapInfo{ID: mindmapID}, model.MindmapFilter{ID: true})
	if err != nil || len(mindmaps) == 0 {
		s.logger.Error(ctx, "Failed to retrieve newly created mindmap", log.Fields{"error": err, "mindmapID": mindmapID})
		return nil, fmt.Errorf("failed to retrieve newly created mindmap: %w", err)
	}
	s.MindmapSet(mindmaps[0])
	s.logger.Debug(ctx, "Set new mindmap as current", log.Fields{"mindmapID": mindmapID})

	s.logger.Info(ctx, "Mindmap added successfully", log.Fields{"mindmapID": mindmapID})
	return mindmapID, nil
}

// handleMindmapDelete handles the mindmap delete command
func handleMindmapDelete(s *Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Handling mindmap delete command", log.Fields{"args": cmd.Args})

	user, err := s.UserGet()
	if err != nil {
		s.logger.Error(ctx, "No user selected", log.Fields{"error": err})
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	if len(cmd.Args) == 0 {
		// Delete current mindmap
		currentMindmap, err := s.MindmapGet()
		if err != nil {
			s.logger.Error(ctx, "No mindmap selected", log.Fields{"error": err})
			return nil, fmt.Errorf("no mindmap selected: %w", err)
		}
		s.logger.Debug(ctx, "Deleting current mindmap", log.Fields{"mindmapID": currentMindmap.ID})
		err = s.DataManager.MindmapManager.MindmapDelete(user, currentMindmap)
		if err != nil {
			s.logger.Error(ctx, "Failed to delete current mindmap", log.Fields{"error": err})
			return nil, fmt.Errorf("failed to delete current mindmap: %w", err)
		}
		s.MindmapSet(nil)
		s.logger.Debug(ctx, "Cleared current mindmap from session", nil)
		s.logger.Info(ctx, "Current mindmap deleted successfully", nil)
		return nil, nil
	}

	// Delete specific mindmap
	mindmapName := cmd.Args[0]
	s.logger.Debug(ctx, "Deleting specific mindmap", log.Fields{"mindmapName": mindmapName})
	mindmaps, err := s.DataManager.MindmapManager.MindmapGet(user, model.MindmapInfo{Name: mindmapName}, model.MindmapFilter{Name: true})
	if err != nil {
		s.logger.Error(ctx, "Failed to get mindmap", log.Fields{"error": err, "mindmapName": mindmapName})
		return nil, fmt.Errorf("failed to get mindmap: %w", err)
	}
	if len(mindmaps) == 0 {
		s.logger.Warn(ctx, "Mindmap not found", log.Fields{"mindmapName": mindmapName})
		return nil, fmt.Errorf("mindmap not found: %s", mindmapName)
	}

	err = s.DataManager.MindmapManager.MindmapDelete(user, mindmaps[0])
	if err != nil {
		s.logger.Error(ctx, "Failed to delete mindmap", log.Fields{"error": err, "mindmapName": mindmapName})
		return nil, fmt.Errorf("failed to delete mindmap: %w", err)
	}

	// If the deleted mindmap was the current one, clear it from the session
	currentMindmap, _ := s.MindmapGet()
	if currentMindmap != nil && currentMindmap.ID == mindmaps[0].ID {
		s.MindmapSet(nil)
		s.logger.Debug(ctx, "Cleared current mindmap from session", nil)
	}

	s.logger.Info(ctx, "Mindmap deleted successfully", log.Fields{"mindmapName": mindmapName})
	return nil, nil
}

// handleMindmapPermission handles the mindmap permission command
func handleMindmapPermission(s *Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Handling mindmap permission command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		s.logger.Error(ctx, "Invalid number of arguments for mindmap permission", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("mindmap permission command requires 1 or 2 arguments: <mindmap_name> [public|private]")
	}

	user, err := s.UserGet()
	if err != nil {
		s.logger.Error(ctx, "No user selected", log.Fields{"error": err})
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	mindmapName := cmd.Args[0]
	s.logger.Debug(ctx, "Getting mindmap for permission change", log.Fields{"mindmapName": mindmapName})
	mindmaps, err := s.DataManager.MindmapManager.MindmapGet(user, model.MindmapInfo{Name: mindmapName}, model.MindmapFilter{Name: true})
	if err != nil {
		s.logger.Error(ctx, "Failed to get mindmap", log.Fields{"error": err, "mindmapName": mindmapName})
		return nil, fmt.Errorf("failed to get mindmap: %w", err)
	}
	if len(mindmaps) == 0 {
		s.logger.Warn(ctx, "Mindmap not found", log.Fields{"mindmapName": mindmapName})
		return nil, fmt.Errorf("mindmap not found: %s", mindmapName)
	}
	mindmap := mindmaps[0]

	if len(cmd.Args) == 1 {
		// Return current permission
		s.logger.Info(ctx, "Returning current mindmap permission", log.Fields{"mindmapName": mindmapName, "isPublic": mindmap.IsPublic})
		return mindmap.IsPublic, nil
	}

	// Set new permission
	isPublic := strings.ToLower(cmd.Args[1]) == "public"
	s.logger.Debug(ctx, "Setting new mindmap permission", log.Fields{"mindmapName": mindmapName, "isPublic": isPublic})
	err = s.DataManager.MindmapManager.MindmapUpdate(user, mindmap, model.MindmapInfo{IsPublic: isPublic}, model.MindmapFilter{IsPublic: true})
	if err != nil {
		s.logger.Error(ctx, "Failed to update mindmap permission", log.Fields{"error": err, "mindmapName": mindmapName})
		return nil, fmt.Errorf("failed to update mindmap permission: %w", err)
	}

	// Update the session's Mindmap object if it's the current mindmap
	currentMindmap, _ := s.MindmapGet()
	if currentMindmap != nil && currentMindmap.ID == mindmap.ID {
		currentMindmap.IsPublic = isPublic
		s.logger.Debug(ctx, "Updated current mindmap permission in session", log.Fields{"isPublic": isPublic})
	}

	s.logger.Info(ctx, "Mindmap permission updated successfully", log.Fields{"mindmapName": mindmapName, "isPublic": isPublic})
	return isPublic, nil
}

// handleMindmapImport handles the mindmap import command
func handleMindmapImport(s *Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Handling mindmap import command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		s.logger.Error(ctx, "Invalid number of arguments for mindmap import", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("mindmap import command requires 1 or 2 arguments: <filename> [json|xml]")
	}

	user, err := s.UserGet()
	if err != nil {
		s.logger.Error(ctx, "No user selected", log.Fields{"error": err})
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	filename := cmd.Args[0]
	format := "json"
	if len(cmd.Args) == 2 {
		format = strings.ToLower(cmd.Args[1])
	}

	if format != "json" && format != "xml" {
		s.logger.Error(ctx, "Invalid import format", log.Fields{"format": format})
		return nil, fmt.Errorf("invalid format: %s. Must be 'json' or 'xml'", format)
	}

	s.logger.Debug(ctx, "Importing mindmap", log.Fields{"filename": filename, "format": format})
	importedMindmap, err := s.DataManager.MindmapImport(user, filename, format)
	if err != nil {
		s.logger.Error(ctx, "Failed to import mindmap", log.Fields{"error": err, "filename": filename})
		return nil, fmt.Errorf("failed to import mindmap: %w", err)
	}

	// Set the imported mindmap as the current mindmap
	s.MindmapSet(importedMindmap)
	s.logger.Debug(ctx, "Set imported mindmap as current", log.Fields{"mindmapID": importedMindmap.ID})

	s.logger.Info(ctx, "Mindmap imported successfully", log.Fields{"mindmapID": importedMindmap.ID, "mindmapName": importedMindmap.Name})
	return importedMindmap, nil
}

// handleMindmapExport handles the mindmap export command
func handleMindmapExport(s *Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Handling mindmap export command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		s.logger.Error(ctx, "Invalid number of arguments for mindmap export", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("mindmap export command requires 1 or 2 arguments: <filename> [json|xml]")
	}

	user, err := s.UserGet()
	if err != nil {
		s.logger.Error(ctx, "No user selected", log.Fields{"error": err})
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	currentMindmap, err := s.MindmapGet()
	if err != nil {
		s.logger.Error(ctx, "No mindmap selected", log.Fields{"error": err})
		return nil, fmt.Errorf("no mindmap selected: %w", err)
	}

	filename := cmd.Args[0]
	format := "json"
	if len(cmd.Args) == 2 {
		format = strings.ToLower(cmd.Args[1])
	}

	if format != "json" && format != "xml" {
		s.logger.Error(ctx, "Invalid export format", log.Fields{"format": format})
		return nil, fmt.Errorf("invalid format: %s. Must be 'json' or 'xml'", format)
	}

	s.logger.Debug(ctx, "Exporting mindmap", log.Fields{"filename": filename, "format": format, "mindmapID": currentMindmap.ID})
	err = s.DataManager.MindmapExport(user, currentMindmap, filename, format)
	if err != nil {
		s.logger.Error(ctx, "Failed to export mindmap", log.Fields{"error": err, "mindmapID": currentMindmap.ID})
		return nil, fmt.Errorf("failed to export mindmap: %w", err)
	}

	s.logger.Info(ctx, "Mindmap exported successfully", log.Fields{"filename": filename, "format": format, "mindmapID": currentMindmap.ID})
	return nil, nil
}

// handleMindmapSelect handles the mindmap select command
func handleMindmapSelect(s *Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Handling mindmap select command", log.Fields{"args": cmd.Args})

	user, err := s.UserGet()
	if err != nil {
		s.logger.Error(ctx, "No user selected", log.Fields{"error": err})
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	if len(cmd.Args) == 0 {
		// Deselect current mindmap
		s.logger.Debug(ctx, "Deselecting current mindmap", nil)
		s.MindmapSet(nil)
		s.logger.Info(ctx, "Current mindmap deselected", nil)
		return nil, nil
	}

	mindmapName := cmd.Args[0]
	s.logger.Debug(ctx, "Selecting mindmap", log.Fields{"mindmapName": mindmapName})
	mindmaps, err := s.DataManager.MindmapManager.MindmapGet(user, model.MindmapInfo{Name: mindmapName}, model.MindmapFilter{Name: true})
	if err != nil {
		s.logger.Error(ctx, "Failed to get mindmap", log.Fields{"error": err, "mindmapName": mindmapName})
		return nil, fmt.Errorf("failed to get mindmap: %w", err)
	}
	if len(mindmaps) == 0 {
		s.logger.Warn(ctx, "Mindmap not found", log.Fields{"mindmapName": mindmapName})
		return nil, fmt.Errorf("mindmap not found: %s", mindmapName)
	}

	selectedMindmap := mindmaps[0]
	s.MindmapSet(selectedMindmap)
	s.logger.Debug(ctx, "Mindmap selected and set in session", log.Fields{"mindmapID": selectedMindmap.ID})

	// Publish MindmapSelected event
	s.DataManager.EventManager.Publish(event.Event{
		Type: event.MindmapSelected,
		Data: selectedMindmap,
	})
	s.logger.Debug(ctx, "Published MindmapSelected event", log.Fields{"mindmapID": selectedMindmap.ID})

	s.logger.Info(ctx, "Mindmap selected successfully", log.Fields{"mindmapName": mindmapName, "mindmapID": selectedMindmap.ID})
	return selectedMindmap, nil
}

// handleMindmapList handles the mindmap list command
func handleMindmapList(s *Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Handling mindmap list command", nil)

	user, err := s.UserGet()
	if err != nil {
		s.logger.Error(ctx, "No user selected", log.Fields{"error": err})
		return nil, fmt.Errorf("no user selected: %w", err)
	}

	s.logger.Debug(ctx, "Retrieving mindmaps for user", log.Fields{"username": user.Username})
	mindmaps, err := s.DataManager.MindmapManager.MindmapGet(user, model.MindmapInfo{}, model.MindmapFilter{})
	if err != nil {
		s.logger.Error(ctx, "Failed to get mindmaps", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to get mindmaps: %w", err)
	}

	s.logger.Info(ctx, "Mindmaps retrieved successfully", log.Fields{"count": len(mindmaps)})
	return mindmaps, nil
}

// handleMindmapView handles the mindmap view command
func handleMindmapView(s *Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	s.logger.Info(ctx, "Handling mindmap view command", log.Fields{"args": cmd.Args})

	currentMindmap, err := s.MindmapGet()
	if err != nil {
		s.logger.Error(ctx, "No mindmap selected", log.Fields{"error": err})
		return nil, fmt.Errorf("no mindmap selected: %w", err)
	}

	showID := false
	var node *model.Node

	for _, arg := range cmd.Args {
		if arg == "--id" {
			showID = true
			s.logger.Debug(ctx, "ID display enabled for mindmap view", nil)
		} else {
			// Assume the argument is an index
			s.logger.Debug(ctx, "Attempting to get node by index", log.Fields{"index": arg})
			nodes, err := s.DataManager.NodeManager.NodeGet(currentMindmap, model.NodeInfo{Index: arg}, model.NodeFilter{Index: true})
			if err != nil {
				s.logger.Error(ctx, "Failed to get node", log.Fields{"error": err, "index": arg})
				return nil, fmt.Errorf("failed to get node: %w", err)
			}
			if len(nodes) == 0 {
				s.logger.Warn(ctx, "Node not found with index", log.Fields{"index": arg})
				return nil, fmt.Errorf("node not found with index: %s", arg)
			}
			node = nodes[0]
		}
	}

	if node == nil {
		node = currentMindmap.Root
		s.logger.Debug(ctx, "Using root node for mindmap view", log.Fields{"nodeID": node.ID})
	}

	// This is a placeholder implementation of formatting node and its children for display
	formattedView := formatNodeForDisplay(node, showID)
	s.logger.Debug(ctx, "Formatted node for display", log.Fields{"nodeID": node.ID})

	s.logger.Info(ctx, "Mindmap view generated successfully", log.Fields{"nodeID": node.ID})
	return formattedView, nil
}

// formatNodeForDisplay is a helper function to format a node and its children for display
// This is a placeholder implementation
func formatNodeForDisplay(node *model.Node, showID bool) string {
	// TODO: remove and implement in node_handler or a different package
	return fmt.Sprintf("Node: %s (ID: %d)", node.Name, node.ID)
}
