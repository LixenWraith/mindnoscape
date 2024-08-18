// Package data provides data management functionality for the Mindnoscape application.
// This file contains the implementation of the history system for undo and redo operations.
package data

import (
	"fmt"

	"mindnoscape/local-app/internal/models"
)

// OperationType defines the types of operations that can be performed on nodes.
type OperationType string

const (
	OpAdd    OperationType = "Add"
	OpDelete OperationType = "Delete"
	OpMove   OperationType = "Move"
	OpUpdate OperationType = "Update"
)

// Operation represents a single operation performed on a node, used for undo/redo functionality.
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

// HistoryManager manages the history of operations for undo and redo functionality.
type HistoryManager struct {
	history      []Operation
	historyIndex int
	nm           *NodeManager
}

// NewHistoryManager creates a new HistoryManager instance.
func NewHistoryManager(nm *NodeManager) *HistoryManager {
	return &HistoryManager{
		history:      []Operation{},
		historyIndex: -1,
		nm:           nm,
	}
}

// HistoryAdd adds a new operation to the history.
func (hm *HistoryManager) HistoryAdd(op Operation) {
	if hm.historyIndex == len(hm.history)-1 {
		hm.history = append(hm.history, op)
	} else {
		hm.history = append(hm.history[:hm.historyIndex+1], op)
	}
	hm.historyIndex++
}

// HistoryReset clears the history.
func (hm *HistoryManager) HistoryReset() {
	hm.history = []Operation{}
	hm.historyIndex = -1
}

// GetLastOperation returns the last operation in the history.
func (hm *HistoryManager) GetLastOperation() (*Operation, error) {
	if hm.historyIndex < 0 {
		return nil, fmt.Errorf("no operations to undo")
	}
	return &hm.history[hm.historyIndex], nil
}

// GetNextOperation returns the next operation in the history for redo.
func (hm *HistoryManager) GetNextOperation() (*Operation, error) {
	if hm.historyIndex >= len(hm.history)-1 {
		return nil, fmt.Errorf("no operations to redo")
	}
	return &hm.history[hm.historyIndex+1], nil
}

// RemoveLastOperation removes the last operation from the history.
func (hm *HistoryManager) RemoveLastOperation() {
	if hm.historyIndex >= 0 {
		hm.historyIndex--
	}
}

// MoveToNextOperation moves the history index to the next operation.
func (hm *HistoryManager) MoveToNextOperation() {
	if hm.historyIndex < len(hm.history)-1 {
		hm.historyIndex++
	}
}
