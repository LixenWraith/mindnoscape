package data

import (
	"fmt"
	"strconv"

	"mindnoscape/local-app/internal/models"
)

type OperationType string

const (
	OpAdd    OperationType = "Add"
	OpDelete OperationType = "Delete"
	OpMove   OperationType = "Move"
	OpUpdate OperationType = "Update"
)

type Operation struct {
	Type         OperationType
	AffectedNode models.NodeInfo
	OldParentID  int               // Used for Move
	NewParentID  int               // Used for Move
	OldContent   string            // Used for Update
	NewContent   string            // Used for Update and Add
	OldExtra     map[string]string // Used for Update
	NewExtra     map[string]string // Used for Update and Add
	DeletedTree  []*models.Node    // Used for Delete to store the entire deleted subtree
}

type HistoryManager struct {
	history      []Operation
	historyIndex int
	nm           *NodeManager
}

func NewHistoryManager(nm *NodeManager) *HistoryManager {
	return &HistoryManager{
		history:      []Operation{},
		historyIndex: -1,
		nm:           nm,
	}
}

func (hm *HistoryManager) HistoryAdd(op Operation) {
	if hm.historyIndex == len(hm.history)-1 {
		hm.history = append(hm.history, op)
	} else {
		hm.history = append(hm.history[:hm.historyIndex+1], op)
	}
	hm.historyIndex++
}

func (hm *HistoryManager) HistoryReset() {
	hm.history = []Operation{}
	hm.historyIndex = -1
}

func (hm *HistoryManager) Undo() error {
	if hm.historyIndex < 0 {
		return fmt.Errorf("nothing to undo")
	}

	op := hm.history[hm.historyIndex]

	var err error
	switch op.Type {
	case OpAdd:

		err = hm.nm.NodeDelete(strconv.Itoa(op.AffectedNode.ID), true)
	case OpDelete:

		err = hm.restoreSubtree(op.DeletedTree)
	case OpMove:

		err = hm.nm.NodeMove(strconv.Itoa(op.AffectedNode.ID), strconv.Itoa(op.OldParentID), true)
	case OpUpdate:
		err = hm.nm.NodeUpdate(strconv.Itoa(op.AffectedNode.ID), op.OldContent, op.OldExtra, true)
	}

	if err != nil {
		return fmt.Errorf("failed to undo %s: %w", op.Type, err)
	}

	hm.historyIndex--
	return nil
}

func (hm *HistoryManager) Redo() error {
	if hm.historyIndex >= len(hm.history)-1 {
		return fmt.Errorf("nothing to redo")
	}

	op := hm.history[hm.historyIndex+1]

	var err error
	switch op.Type {
	case OpAdd:

		err = hm.nm.NodeAdd(strconv.Itoa(op.AffectedNode.ParentID), op.NewContent, op.NewExtra, true)
	case OpDelete:

		err = hm.nm.NodeDelete(strconv.Itoa(op.AffectedNode.ID), true)
	case OpMove:

		err = hm.nm.NodeMove(strconv.Itoa(op.AffectedNode.ID), strconv.Itoa(op.NewParentID), true)
	case OpUpdate:
		err = hm.nm.NodeUpdate(strconv.Itoa(op.AffectedNode.ID), op.NewContent, op.NewExtra, true)
	}

	if err != nil {
		return fmt.Errorf("failed to redo %s: %w", op.Type, err)
	}

	hm.historyIndex++
	return nil
}

func (hm *HistoryManager) restoreSubtree(nodes []*models.Node) error {
	for _, node := range nodes {
		err := hm.nm.NodeAdd(strconv.Itoa(node.ParentID), node.Content, node.Extra, true)
		if err != nil {
			return fmt.Errorf("failed to restore node %d: %w", node.Index, err)
		}
	}
	return nil
}
