package models

type Node struct {
	Index        int               `json:"index" xml:"index,attr"`
	ParentID     int               `json:"parent_id" xml:"parent_id,attr"`
	Content      string            `json:"content" xml:"content"`
	Children     []*Node           `json:"children,omitempty" xml:"children>node,omitempty"`
	Extra        map[string]string `json:"extra,omitempty" xml:"extra>field,omitempty"`
	LogicalIndex string            `json:"logical_index" xml:"logical_index,attr"`
	MindMapID    int               `json:"mindmap_id" xml:"mindmap_id,attr"` // New field
}

func NewNode(index int, content string, mindMapID int) *Node {
	return &Node{
		Index:     index,
		Content:   content,
		Children:  make([]*Node, 0),
		Extra:     make(map[string]string),
		MindMapID: mindMapID,
	}
}
