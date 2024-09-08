// Package model defines the data structures used throughout the Mindnoscape application.
package model

import "time"

// Node represents a single node in a mind map.
type Node struct {
	ID        int               `json:"id" xml:"id,attr"`
	MindmapID int               `json:"mindmap_id" xml:"mindmap_id,attr"`
	ParentID  int               `json:"parent_id" xml:"parent_id,attr"`
	Name      string            `json:"name" xml:"name,attr"`
	Index     string            `json:"index" xml:"index,attr"`
	Content   map[string]string `json:"content,omitempty" xml:"content,omitempty"`
	Children  []*Node           `json:"children,omitempty" xml:"children>node,omitempty"`
	Created   time.Time         `json:"created" xml:"created,attr"`
	Updated   time.Time         `json:"updated" xml:"updated,attr"`
}

// NodeInfo contains basic information about a node.
type NodeInfo struct {
	ID        int
	MindmapID int
	ParentID  int
	Name      string
	Index     string
	Content   map[string]string
}

// NodeFilter defines the options for filtering nodes.
type NodeFilter struct {
	ID        bool
	MindmapID bool
	ParentID  bool
	Name      bool
	Index     bool
	Content   bool
}
