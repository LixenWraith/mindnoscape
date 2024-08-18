// Package ui provides user interface functionality for the Mindnoscape application.
// This file contains the UserUI struct and methods for displaying user-related information.
package ui

import (
	"io"
)

// UserUI handles the visualization of user-related information.
type UserUI struct {
	visualizer *Visualizer
}

// MindmapInfo contains basic information about a mindmap for listing purposes.
type MindmapInfo struct {
	ID       int
	Name     string
	IsPublic bool
	Owner    string
}

// NewUserUI creates a new UserUI instance.
func NewUserUI(w io.Writer, useColor bool) *UserUI {
	return &UserUI{
		visualizer: NewVisualizer(w, useColor),
	}
}
