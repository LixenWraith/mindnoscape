package session

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"mindnoscape/local-app/pkg/model"
)

func handleNodeAdd(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) < 2 {
		return nil, errors.New("node add command requires at least 2 arguments: <parent> <content> [<extra field label>:<extra field value>]... [--id]")
	}

	currentMindmap, err := s.MindmapGet()
	if err != nil {
		return nil, fmt.Errorf("no mindmap selected: %w", err)
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

	parentNode, err := getNode(s, currentMindmap, parentIdentifier, useID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent node: %w", err)
	}

	newNode := model.NodeInfo{
		MindmapID: currentMindmap.ID,
		ParentID:  parentNode.ID,
		Name:      content,
		Content:   extraFields,
	}

	nodeID, _, err := s.DataManager.NodeManager.NodeAdd(currentMindmap, newNode)
	if err != nil {
		return nil, fmt.Errorf("failed to add node: %w", err)
	}

	return nodeID, nil
}

func handleNodeUpdate(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) < 2 {
		return nil, errors.New("node update command requires at least 2 arguments: <node> <content> [<extra field label>:<extra field value>]... [--id]")
	}

	currentMindmap, err := s.MindmapGet()
	if err != nil {
		return nil, fmt.Errorf("no mindmap selected: %w", err)
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

	node, err := getNode(s, currentMindmap, nodeIdentifier, useID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	updateInfo := model.NodeInfo{
		Name:    content,
		Content: extraFields,
	}

	err = s.DataManager.NodeManager.NodeUpdate(currentMindmap, node, updateInfo, model.NodeFilter{Name: true, Content: true})
	if err != nil {
		return nil, fmt.Errorf("failed to update node: %w", err)
	}

	return nil, nil
}

func handleNodeMove(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) < 2 || len(cmd.Args) > 3 {
		return nil, errors.New("node move command requires 2 or 3 arguments: <source> <target> [--id]")
	}

	currentMindmap, err := s.MindmapGet()
	if err != nil {
		return nil, fmt.Errorf("no mindmap selected: %w", err)
	}

	sourceIdentifier := cmd.Args[0]
	targetIdentifier := cmd.Args[1]
	useID := len(cmd.Args) == 3 && cmd.Args[2] == "--id"

	sourceNode, err := getNode(s, currentMindmap, sourceIdentifier, useID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source node: %w", err)
	}

	targetNode, err := getNode(s, currentMindmap, targetIdentifier, useID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target node: %w", err)
	}

	updateInfo := model.NodeInfo{
		ParentID: targetNode.ID,
	}

	err = s.DataManager.NodeManager.NodeUpdate(currentMindmap, sourceNode, updateInfo, model.NodeFilter{ParentID: true})
	if err != nil {
		return nil, fmt.Errorf("failed to move node: %w", err)
	}

	return nil, nil
}

func handleNodeDelete(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		return nil, errors.New("node delete command requires 1 or 2 arguments: <node> [--id]")
	}

	currentMindmap, err := s.MindmapGet()
	if err != nil {
		return nil, fmt.Errorf("no mindmap selected: %w", err)
	}

	nodeIdentifier := cmd.Args[0]
	useID := len(cmd.Args) == 2 && cmd.Args[1] == "--id"

	node, err := getNode(s, currentMindmap, nodeIdentifier, useID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	err = s.DataManager.NodeManager.NodeDelete(currentMindmap, node)
	if err != nil {
		return nil, fmt.Errorf("failed to delete node: %w", err)
	}

	return nil, nil
}

func handleNodeFind(s *Session, cmd model.Command) (interface{}, error) {
	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		return nil, errors.New("node find command requires 1 or 2 arguments: <query> [--id]")
	}

	currentMindmap, err := s.MindmapGet()
	if err != nil {
		return nil, fmt.Errorf("no mindmap selected: %w", err)
	}

	query := cmd.Args[0]
	showID := len(cmd.Args) == 2 && cmd.Args[1] == "--id"

	nodes, err := s.DataManager.NodeManager.NodeFind(currentMindmap, model.NodeFilter{Name: true, Content: true}, query)
	if err != nil {
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

	return results, nil
}

func handleNodeSort(s *Session, cmd model.Command) (interface{}, error) {
	currentMindmap, err := s.MindmapGet()
	if err != nil {
		return nil, fmt.Errorf("no mindmap selected: %w", err)
	}

	var parentNode *model.Node
	var field string
	reverse := false
	useID := false

	for i, arg := range cmd.Args {
		switch {
		case i == 0 && arg != "--reverse" && arg != "--id":
			parentNode, err = getNode(s, currentMindmap, arg, useID)
			if err != nil {
				return nil, fmt.Errorf("failed to get parent node: %w", err)
			}
		case arg == "--reverse":
			reverse = true
		case arg == "--id":
			useID = true
		default:
			field = arg
		}
	}

	if parentNode == nil {
		parentNode = currentMindmap.Root
	}

	err = s.DataManager.NodeManager.NodeSort(currentMindmap, s.DataManager.NodeManager.NodeToInfo(parentNode), field, reverse)
	if err != nil {
		return nil, fmt.Errorf("failed to sort nodes: %w", err)
	}

	return nil, nil
}

// getNode is a helper function to get a node by its identifier (index or ID)
func getNode(s *Session, mindmap *model.Mindmap, identifier string, useID bool) (*model.Node, error) {
	var nodeInfo model.NodeInfo
	var nodeFilter model.NodeFilter

	if useID {
		id, err := strconv.Atoi(identifier)
		if err != nil {
			return nil, fmt.Errorf("invalid node ID: %s", identifier)
		}
		nodeInfo.ID = id
		nodeFilter.ID = true
	} else {
		nodeInfo.Index = identifier
		nodeFilter.Index = true
	}

	nodes, err := s.DataManager.NodeManager.NodeGet(mindmap, nodeInfo, nodeFilter)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("node not found: %s", identifier)
	}
	return nodes[0], nil
}
