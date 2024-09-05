// Package model defines the data structures used throughout the Mindnoscape application.
package model

import "time"

// Mindmap represents a complete mind map structure with its nodes and metadata.
type Mindmap struct {
	ID       int           `json:"id" xml:"id,attr"`
	Name     string        `json:"name" xml:"name,attr"`
	Owner    string        `json:"owner" xml:"owner,attr"`
	IsPublic bool          `json:"is_public" xml:"is_public,attr"`
	Root     *Node         `json:"root" xml:"root"`
	Nodes    map[int]*Node `json:"nodes,omitempty" xml:"nodes>node,omitempty"`
	Created  time.Time     `json:"created" xml:"created,attr"`
	Updated  time.Time     `json:"updated" xml:"updated,attr"`
}

// MindmapInfo contains basic information about a mindmap.
type MindmapInfo struct {
	ID        int
	Name      string
	Owner     string
	IsPublic  bool
	NodeCount *int
	Depth     *int
}

// MindmapFilter defines the options for filtering mindmap data.
type MindmapFilter struct {
	ID       bool
	Name     bool
	Owner    bool
	IsPublic bool
}
