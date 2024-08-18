// Package models defines the data structures used throughout the Mindnoscape application.
package models

// Mindmap represents a complete mind map structure with its nodes and metadata.
type Mindmap struct {
	ID       int           `json:"-" xml:"-"`
	Name     string        `json:"name" xml:"name,attr"`
	Owner    string        `json:"-" xml:"-"`
	IsPublic bool          `json:"-" xml:"-"`
	Root     *Node         `json:"root" xml:"root"`
	Nodes    map[int]*Node `json:"-" xml:"-"`
}

// MindmapInfo contains basic information about a mindmap.
type MindmapInfo struct {
	ID       int
	Name     string
	IsPublic bool
	Owner    string
}

// NewMindmap creates a new Mindmap instance with initialized fields.
func NewMindmap(id int, name string, owner string, isPublic bool) *Mindmap {
	return &Mindmap{
		ID:       id,
		Name:     name,
		Owner:    owner,
		IsPublic: isPublic,
		Root:     &Node{ID: 0, ParentID: -1, Content: name, Index: "0"},
		Nodes:    make(map[int]*Node),
	}
}
