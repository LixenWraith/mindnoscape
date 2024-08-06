package mindmap

import (
	"fmt"
	"sort"
	"strings"
	"strconv"

	"mindnoscape/local-app/internal/models"
)

func (mm *MindMap) AddNode(parentIdentifier string, content string, extra map[string]string, useIndex bool) error {
    fmt.Printf("DEBUG: AddNode called with parentIdentifier: %s, content: %s, useIndex: %v\n", parentIdentifier, content, useIndex)

    var parentNode *models.Node
    if useIndex {
        index, err := strconv.Atoi(parentIdentifier)
        if err != nil {
            return fmt.Errorf("invalid index: %v", err)
        }
        parentNode = mm.Nodes[index]
        fmt.Printf("DEBUG: Using index. Parent node found: %v\n", parentNode != nil)
    } else {
        if parentIdentifier == "0" {
            parentNode = mm.Root
            fmt.Printf("DEBUG: Using root as parent node\n")
        } else {
            parentNode = mm.findNodeByLogicalIndex(parentIdentifier)
            fmt.Printf("DEBUG: Using logical index. Parent node found: %v\n", parentNode != nil)
        }
    }

    if parentNode == nil {
        fmt.Printf("DEBUG: Parent node is nil. Nodes in mm.Nodes: %d\n", len(mm.Nodes))
        for k, v := range mm.Nodes {
            fmt.Printf("DEBUG: Node[%d] = %+v\n", k, v)
        }
        return fmt.Errorf("parent node not found")
    }

    newIndex := mm.getNextIndex()
    fmt.Printf("DEBUG: New node index: %d\n", newIndex)

    newNode := models.NewNode(newIndex, content)
    newNode.Extra = extra
    newNode.ParentID = parentNode.Index

    // Assign logical index
    if parentNode == mm.Root {
        newNode.LogicalIndex = fmt.Sprintf("%d", len(parentNode.Children)+1)
    } else {
        newNode.LogicalIndex = fmt.Sprintf("%s.%d", parentNode.LogicalIndex, len(parentNode.Children)+1)
    }

    // Ensure root node has a logical index
    if mm.Root.LogicalIndex == "" {
        mm.Root.LogicalIndex = "0"
        // Update root node in storage
        if err := mm.Store.UpdateNode(mm.Root.Index, mm.Root.Content, mm.Root.Extra, mm.Root.LogicalIndex); err != nil {
            return fmt.Errorf("failed to update root node in storage: %v", err)
        }
    }

    // Add to storage
    if err := mm.Store.AddNode(parentNode.Index, newNode.Content, newNode.Extra, newNode.LogicalIndex); err != nil {
        return fmt.Errorf("failed to add node to storage: %v", err)
    }

    // Update in-memory structure
    parentNode.Children = append(parentNode.Children, newNode)
    mm.Nodes[newIndex] = newNode

    fmt.Printf("DEBUG: Node added successfully. New node: %+v\n", newNode)

    return nil
}

func (mm *MindMap) getNextIndex() int {
    mm.MaxIndex++
    return mm.MaxIndex
}

func (mm *MindMap) DeleteNode(identifier string, useIndex bool) error {
	var nodeToDelete *models.Node
	if useIndex {
		index, err := strconv.Atoi(identifier)
		if err != nil {
			return fmt.Errorf("invalid index: %v", err)
		}
		nodeToDelete = mm.Nodes[index]
	} else {
		nodeToDelete = mm.findNodeByLogicalIndex(identifier)
	}

	if nodeToDelete == nil {
		return fmt.Errorf("node not found")
	}

	if nodeToDelete == mm.Root {
		return fmt.Errorf("cannot delete root node")
	}

	// Remove from storage
	if err := mm.Store.DeleteNode(nodeToDelete.Index); err != nil {
		return fmt.Errorf("failed to delete node from storage: %v", err)
	}

	// Update in-memory structure
	for _, parentNode := range mm.Nodes {
		for i, child := range parentNode.Children {
			if child == nodeToDelete {
				parentNode.Children = append(parentNode.Children[:i], parentNode.Children[i+1:]...)
				break
			}
		}
	}
	mm.deleteNodeRecursive(nodeToDelete)
	mm.assignLogicalIndex(mm.Root, "")

	return nil
}

func (mm *MindMap) deleteNodeRecursive(node *models.Node) {
	for _, child := range node.Children {
		mm.deleteNodeRecursive(child)
	}
	delete(mm.Nodes, node.Index)
}

func (mm *MindMap) ModifyNode(identifier string, content string, extra map[string]string, useIndex bool) error {
	var node *models.Node
	if useIndex {
		index, err := strconv.Atoi(identifier)
		if err != nil {
			return fmt.Errorf("invalid index: %v", err)
		}
		node = mm.Nodes[index]
	} else {
		node = mm.findNodeByLogicalIndex(identifier)
	}

	if node == nil {
		return fmt.Errorf("node not found")
	}

	if content != "" {
		node.Content = content
	}
	for k, v := range extra {
		if v == "" {
			delete(node.Extra, k)
		} else {
			node.Extra[k] = v
		}
	}

	// Update in storage
    if err := mm.Store.UpdateNode(node.Index, node.Content, node.Extra, node.LogicalIndex); err != nil {
        return fmt.Errorf("failed to update node in storage: %v", err)
    }

	return nil
}

func (mm *MindMap) MoveNode(sourceIdentifier, targetIdentifier string, useIndex bool) error {
	var sourceNode, targetNode *models.Node
	if useIndex {
		sourceIndex, err := strconv.Atoi(sourceIdentifier)
		if err != nil {
			return fmt.Errorf("invalid source index: %v", err)
		}
		sourceNode = mm.Nodes[sourceIndex]

		targetIndex, err := strconv.Atoi(targetIdentifier)
		if err != nil {
			return fmt.Errorf("invalid target index: %v", err)
		}
		targetNode = mm.Nodes[targetIndex]
	} else {
		sourceNode = mm.findNodeByLogicalIndex(sourceIdentifier)
		targetNode = mm.findNodeByLogicalIndex(targetIdentifier)
	}

	if sourceNode == nil || targetNode == nil {
		return fmt.Errorf("source or target node not found")
	}

	if sourceNode == mm.Root {
		return fmt.Errorf("cannot move root node")
	}

	// Remove from old parent
	for _, parentNode := range mm.Nodes {
		for i, child := range parentNode.Children {
			if child == sourceNode {
				parentNode.Children = append(parentNode.Children[:i], parentNode.Children[i+1:]...)
				break
			}
		}
	}

	// Add to new parent
	targetNode.Children = append(targetNode.Children, sourceNode)

	// Update in storage
	if err := mm.Store.MoveNode(sourceNode.Index, targetNode.Index); err != nil {
		return fmt.Errorf("failed to move node in storage: %v", err)
	}

	mm.assignLogicalIndex(mm.Root, "")
	return nil
}

func (mm *MindMap) Sort(identifier string, field string, reverse bool, useIndex bool) error {
    fmt.Printf("DEBUG: Sort called with identifier: %s, field: %s, reverse: %v, useIndex: %v\n", identifier, field, reverse, useIndex)

    var node *models.Node
    if identifier == "" {
        node = mm.Root
    } else if useIndex {
        index, err := strconv.Atoi(identifier)
        if err != nil {
            return fmt.Errorf("invalid index: %v", err)
        }
        node = mm.Nodes[index]
    } else {
        node = mm.findNodeByLogicalIndex(identifier)
    }

    if node == nil {
        return fmt.Errorf("node not found")
    }

    fmt.Printf("DEBUG: Sorting children of node: %s (Index: %d)\n", node.Content, node.Index)
    fmt.Printf("DEBUG: Children before sort: %v\n", nodesToString(node.Children))

    mm.sortNodeChildrenRecursively(node, field, reverse)
    mm.assignLogicalIndexForSubtree(node)

    // Update the database to reflect the new order
    err := mm.updateNodeOrderInDB(node)
    if err != nil {
        return fmt.Errorf("failed to update node order in database: %v", err)
    }

    // Reload nodes to ensure in-memory structure is in sync with the database
    err = mm.loadNodes()
    if err != nil {
        return fmt.Errorf("failed to reload nodes after sorting: %v", err)
    }

    return nil
}

func (mm *MindMap) assignLogicalIndexForSubtree(node *models.Node) {
    fmt.Printf("DEBUG: assignLogicalIndexForSubtree called for node: %s\n", node.Content)

    var assign func(*models.Node, string)
    assign = func(n *models.Node, prefix string) {
        for i, child := range n.Children {
            if n == mm.Root {
                child.LogicalIndex = fmt.Sprintf("%d", i+1)
            } else {
                child.LogicalIndex = fmt.Sprintf("%s.%d", prefix, i+1)
            }
            fmt.Printf("DEBUG: Assigned LogicalIndex %s to node %s\n", child.LogicalIndex, child.Content)
            assign(child, child.LogicalIndex)
        }
    }

    if node == mm.Root {
        assign(node, "")
    } else {
        assign(node, node.LogicalIndex)
    }
}

func nodesToString(nodes []*models.Node) string {
    var s []string
    for _, n := range nodes {
        s = append(s, fmt.Sprintf("{Content: %s, LogicalIndex: %s}", n.Content, n.LogicalIndex))
    }
    return strings.Join(s, ", ")
}

func (mm *MindMap) sortNodeChildrenRecursively(node *models.Node, field string, reverse bool) {
    fmt.Printf("DEBUG: sortNodeChildrenRecursively called for node: %s, field: %s, reverse: %v\n", node.Content, field, reverse)

    sort.Slice(node.Children, func(i, j int) bool {
        var vi, vj string
        if field == "" {
            vi, vj = node.Children[i].Content, node.Children[j].Content
        } else {
            vi = node.Children[i].Extra[field]
            vj = node.Children[j].Extra[field]
        }

        // If the field doesn't exist, fall back to Content
        if vi == "" && vj == "" {
            vi, vj = node.Children[i].Content, node.Children[j].Content
        }

        // Try to compare as numbers if possible
        ni, errI := strconv.ParseFloat(vi, 64)
        nj, errJ := strconv.ParseFloat(vj, 64)
        if errI == nil && errJ == nil {
            if reverse {
                return ni > nj
            }
            return ni < nj
        }

        // Fall back to string comparison
        if reverse {
            return vi > vj
        }
        return vi < vj
    })

    fmt.Printf("DEBUG: Children after sorting: %v\n", nodesToString(node.Children))

    // Recursively sort children of children
    for _, child := range node.Children {
        mm.sortNodeChildrenRecursively(child, field, reverse)
    }
}

func (mm *MindMap) updateNodeOrderInDB(node *models.Node) error {
    fmt.Printf("DEBUG: updateNodeOrderInDB called for node: %s\n", node.Content)

    var updateNode func(*models.Node) error
    updateNode = func(n *models.Node) error {
        for _, child := range n.Children {
            fmt.Printf("DEBUG: Updating order for child %s (Index: %d), LogicalIndex: %s\n", child.Content, child.Index, child.LogicalIndex)

            err := mm.Store.UpdateNodeOrder(child.Index, child.LogicalIndex)
            if err != nil {
                return err
            }

            // Recursively update the order of grandchildren
            err = updateNode(child)
            if err != nil {
                return err
            }
        }
        return nil
    }

    return updateNode(node)
}
