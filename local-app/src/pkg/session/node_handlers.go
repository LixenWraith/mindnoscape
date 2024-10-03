package session

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
)

// handleNodeAdd handles the node add command
func handleNodeAdd(sm *SessionManager, session *model.Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Handling node add command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) < 2 {
		sm.logger.Error(ctx, "Insufficient arguments for node add", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("node add command requires at least 2 arguments: <parent> <content> [<extra field label>:<extra field value>]... [--id]")
	}

	if session.Mindmap == nil {
		sm.logger.Error(ctx, "No mindmap selected", nil)
		return nil, fmt.Errorf("no mindmap selected")
	}

	parentIdentifier := cmd.Args[0]
	content := cmd.Args[1]
	extraFields := make(map[string]string)
	useID := false

	for _, arg := range cmd.Args[2:] {
		if arg == "--id" {
			useID = true
		} else if strings.Contains(arg, ":") {
			parts := strings.SplitN(arg, ":", 2)
			extraFields[parts[0]] = parts[1]
		}
	}

	sm.logger.Debug(ctx, "Parsing node add arguments", log.Fields{"parentIdentifier": parentIdentifier, "content": content, "useID": useID, "extraFields": extraFields})

	parentNode, err := getNode(sm, session.Mindmap, parentIdentifier, useID)
	if err != nil {
		sm.logger.Error(ctx, "Failed to get parent node", log.Fields{"error": err, "parentIdentifier": parentIdentifier})
		return nil, fmt.Errorf("failed to get parent node: %w", err)
	}

	newNode := model.NodeInfo{
		MindmapID: session.Mindmap.ID,
		ParentID:  parentNode.ID,
		Name:      content,
		Content:   extraFields,
	}

	sm.logger.Debug(ctx, "Adding new node", log.Fields{"parentID": parentNode.ID, "content": content})
	nodeID, _, err := sm.dataManager.NodeManager.NodeAdd(session.Mindmap, newNode)
	if err != nil {
		sm.logger.Error(ctx, "Failed to add node", log.Fields{"error": err})
		return nil, fmt.Errorf("failed to add node: %w", err)
	}

	sm.logger.Info(ctx, "Node added successfully", log.Fields{"nodeID": nodeID})
	return nodeID, nil
}

// handleNodeUpdate handles the node update command
func handleNodeUpdate(sm *SessionManager, session *model.Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Handling node update command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) < 2 {
		sm.logger.Error(ctx, "Insufficient arguments for node update", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("node update command requires at least 2 arguments: <node> <content> [<extra field label>:<extra field value>]... [--id]")
	}

	if session.Mindmap == nil {
		sm.logger.Error(ctx, "No mindmap selected", nil)
		return nil, fmt.Errorf("no mindmap selected")
	}

	nodeIdentifier := cmd.Args[0]
	content := cmd.Args[1]
	extraFields := make(map[string]string)
	useID := false

	for _, arg := range cmd.Args[2:] {
		if arg == "--id" {
			useID = true
		} else if strings.Contains(arg, ":") {
			parts := strings.SplitN(arg, ":", 2)
			extraFields[parts[0]] = parts[1]
		}
	}

	sm.logger.Debug(ctx, "Parsing node update arguments", log.Fields{"nodeIdentifier": nodeIdentifier, "content": content, "useID": useID, "extraFields": extraFields})

	node, err := getNode(sm, session.Mindmap, nodeIdentifier, useID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	updateInfo := model.NodeInfo{
		Name:    content,
		Content: extraFields,
	}

	sm.logger.Debug(ctx, "Updating node", log.Fields{"nodeID": node.ID, "newContent": content})
	err = sm.dataManager.NodeManager.NodeUpdate(session.Mindmap, node, updateInfo, model.NodeFilter{Name: true, Content: true})
	if err != nil {
		sm.logger.Error(ctx, "Failed to update node", log.Fields{"error": err, "nodeID": node.ID})
		return nil, fmt.Errorf("failed to update node: %w", err)
	}

	sm.logger.Info(ctx, "Node updated successfully", log.Fields{"nodeID": node.ID})
	return nil, nil
}

// handleNodeMove handles the node move command
func handleNodeMove(sm *SessionManager, session *model.Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Handling node move command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) < 2 || len(cmd.Args) > 3 {
		sm.logger.Error(ctx, "Invalid number of arguments for node move", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("node move command requires 2 or 3 arguments: <source> <target> [--id]")
	}

	if session.Mindmap == nil {
		sm.logger.Error(ctx, "No mindmap selected", nil)
		return nil, fmt.Errorf("no mindmap selected")
	}

	sourceIdentifier := cmd.Args[0]
	targetIdentifier := cmd.Args[1]
	useID := len(cmd.Args) == 3 && cmd.Args[2] == "--id"

	sm.logger.Debug(ctx, "Parsing node move arguments", log.Fields{"sourceIdentifier": sourceIdentifier, "targetIdentifier": targetIdentifier, "useID": useID})

	sourceNode, err := getNode(sm, session.Mindmap, sourceIdentifier, useID)
	if err != nil {
		sm.logger.Error(ctx, "Failed to get source node", log.Fields{"error": err, "sourceIdentifier": sourceIdentifier})
		return nil, fmt.Errorf("failed to get source node: %w", err)
	}

	targetNode, err := getNode(sm, session.Mindmap, targetIdentifier, useID)
	if err != nil {
		sm.logger.Error(ctx, "Failed to get target node", log.Fields{"error": err, "targetIdentifier": targetIdentifier})
		return nil, fmt.Errorf("failed to get target node: %w", err)
	}

	updateInfo := model.NodeInfo{
		ParentID: targetNode.ID,
	}

	sm.logger.Debug(ctx, "Moving node", log.Fields{"sourceNodeID": sourceNode.ID, "targetNodeID": targetNode.ID})
	err = sm.dataManager.NodeManager.NodeUpdate(session.Mindmap, sourceNode, updateInfo, model.NodeFilter{ParentID: true})
	if err != nil {
		sm.logger.Error(ctx, "Failed to move node", log.Fields{"error": err, "sourceNodeID": sourceNode.ID, "targetNodeID": targetNode.ID})
		return nil, fmt.Errorf("failed to move node: %w", err)
	}

	sm.logger.Info(ctx, "Node moved successfully", log.Fields{"sourceNodeID": sourceNode.ID, "targetNodeID": targetNode.ID})
	return nil, nil
}

// handleNodeDelete handles the node delete command
func handleNodeDelete(sm *SessionManager, session *model.Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Handling node delete command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		sm.logger.Error(ctx, "Invalid number of arguments for node delete", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("node delete command requires 1 or 2 arguments: <node> [--id]")
	}

	if session.Mindmap == nil {
		sm.logger.Error(ctx, "No mindmap selected", nil)
		return nil, fmt.Errorf("no mindmap selected")
	}

	nodeIdentifier := cmd.Args[0]
	useID := len(cmd.Args) == 2 && cmd.Args[1] == "--id"

	sm.logger.Debug(ctx, "Parsing node delete arguments", log.Fields{"nodeIdentifier": nodeIdentifier, "useID": useID})

	node, err := getNode(sm, session.Mindmap, nodeIdentifier, useID)
	if err != nil {
		sm.logger.Error(ctx, "Failed to get node", log.Fields{"error": err, "nodeIdentifier": nodeIdentifier})
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	sm.logger.Debug(ctx, "Deleting node", log.Fields{"nodeID": node.ID})
	err = sm.dataManager.NodeManager.NodeDelete(session.Mindmap, node)
	if err != nil {
		sm.logger.Error(ctx, "Failed to delete node", log.Fields{"error": err, "nodeID": node.ID})
		return nil, fmt.Errorf("failed to delete node: %w", err)
	}

	sm.logger.Info(ctx, "Node deleted successfully", log.Fields{"nodeID": node.ID})
	return nil, nil
}

// handleNodeFind handles the node find command
func handleNodeFind(sm *SessionManager, session *model.Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Handling node find command", log.Fields{"args": cmd.Args})

	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		sm.logger.Error(ctx, "Invalid number of arguments for node find", log.Fields{"argCount": len(cmd.Args)})
		return nil, errors.New("node find command requires 1 or 2 arguments: <query> [--id]")
	}

	if session.Mindmap == nil {
		sm.logger.Error(ctx, "No mindmap selected", nil)
		return nil, fmt.Errorf("no mindmap selected")
	}

	query := cmd.Args[0]
	showID := len(cmd.Args) == 2 && cmd.Args[1] == "--id"

	sm.logger.Debug(ctx, "Searching for nodes", log.Fields{"query": query, "showID": showID})
	nodes, err := sm.dataManager.NodeManager.NodeFind(session.Mindmap, model.NodeFilter{Name: true, Content: true}, query)
	if err != nil {
		sm.logger.Error(ctx, "Failed to find nodes", log.Fields{"error": err, "query": query})
		return nil, fmt.Errorf("failed to find nodes: %w", err)
	}

	// Format the results
	var results []string
	for _, node := range nodes {
		if showID {
			results = append(results, fmt.Sprintf("ID: %d, Name: %s, Index: %s", node.ID, node.Name, node.Index))
		} else {
			results = append(results, fmt.Sprintf("Name: %s, Index: %s", node.Name, node.Index))
		}
	}

	sm.logger.Info(ctx, "Nodes found", log.Fields{"count": len(nodes)})
	return results, nil
}

// handleNodeSort handles the node sort command
func handleNodeSort(sm *SessionManager, session *model.Session, cmd model.Command) (interface{}, error) {
	ctx := context.Background()
	sm.logger.Info(ctx, "Handling node sort command", log.Fields{"args": cmd.Args})

	if session.Mindmap == nil {
		sm.logger.Error(ctx, "No mindmap selected", nil)
		return nil, fmt.Errorf("no mindmap selected")
	}

	var parentNode *model.Node
	var field string
	reverse := false
	useID := false
	var parentIdentifier string

	for i, arg := range cmd.Args {
		switch {
		case i == 0 && arg != "--reverse" && arg != "--id":
			parentIdentifier = arg
		case arg == "--reverse":
			reverse = true
		case arg == "--id":
			useID = true
		default:
			field = arg
		}
	}

	if parentIdentifier != "" {
		var err error
		parentNode, err = getNode(sm, session.Mindmap, parentIdentifier, useID)
		if err != nil {
			sm.logger.Error(ctx, "Failed to get parent node", log.Fields{"error": err, "parentIdentifier": parentIdentifier})
			return nil, fmt.Errorf("failed to get parent node: %w", err)
		}
	} else {
		parentNode = session.Mindmap.Root
	}

	sm.logger.Debug(ctx, "Sorting nodes", log.Fields{"parentNodeID": parentNode.ID, "field": field, "reverse": reverse})
	err := sm.dataManager.NodeManager.NodeSort(session.Mindmap, sm.dataManager.NodeManager.NodeToInfo(parentNode), field, reverse)
	if err != nil {
		sm.logger.Error(ctx, "Failed to sort nodes", log.Fields{"error": err, "parentNodeID": parentNode.ID})
		return nil, fmt.Errorf("failed to sort nodes: %w", err)
	}

	sm.logger.Info(ctx, "Nodes sorted successfully", log.Fields{"parentNodeID": parentNode.ID})
	return nil, nil
}

// getNode is a helper function to get a node by its identifier (index or ID)
func getNode(sm *SessionManager, mindmap *model.Mindmap, identifier string, useID bool) (*model.Node, error) {
	ctx := context.Background()
	sm.logger.Debug(ctx, "Getting node", log.Fields{"identifier": identifier, "useID": useID})

	var nodeInfo model.NodeInfo
	var nodeFilter model.NodeFilter

	if useID {
		id, err := strconv.Atoi(identifier)
		if err != nil {
			sm.logger.Error(ctx, "Invalid node ID", log.Fields{"identifier": identifier, "error": err})
			return nil, fmt.Errorf("invalid node ID: %s", identifier)
		}
		nodeInfo.ID = id
		nodeFilter.ID = true
	} else {
		nodeInfo.Index = identifier
		nodeFilter.Index = true
	}

	nodes, err := sm.dataManager.NodeManager.NodeGet(mindmap, nodeInfo, nodeFilter)
	if err != nil {
		sm.logger.Error(ctx, "Failed to get node", log.Fields{"error": err, "identifier": identifier})
		return nil, err
	}
	if len(nodes) == 0 {
		sm.logger.Warn(ctx, "Node not found", log.Fields{"identifier": identifier})
		return nil, fmt.Errorf("node not found: %s", identifier)
	}
	sm.logger.Debug(ctx, "Node retrieved successfully", log.Fields{"nodeID": nodes[0].ID})
	return nodes[0], nil
}
