package models

type Node struct {
	Index        int               `json:"index"`
	ParentID     int               `json:"parent_id"`
	Content      string            `json:"content"`
	Children     []*Node           `json:"children,omitempty"`
	Extra        map[string]string `json:"extra,omitempty"`
	LogicalIndex string            `json:"logical_index"`
}

func NewNode(index int, content string) *Node {
	return &Node{
		Index:    index,
		Content:  content,
		Children: make([]*Node, 0),
		Extra:    make(map[string]string),
	}
}
