package models

type Node struct {
	ID        int               `json:"id" xml:"id,attr"`
	ParentID  int               `json:"parent_id" xml:"parent_id,attr"`
	Content   string            `json:"content" xml:"content"`
	Children  []*Node           `json:"children,omitempty" xml:"children>node,omitempty"`
	Extra     map[string]string `json:"extra,omitempty" xml:"extra>field,omitempty"`
	Index     string            `json:"index" xml:"index,attr"`
	MindmapID int               `json:"mindmap_id" xml:"mindmap_id,attr"`
}

type NodeInfo struct {
	ID       int
	ParentID int
}

func NewNode(id int, content string, mindMapID int) *Node {
	return &Node{
		ID:        id,
		Content:   content,
		Children:  make([]*Node, 0),
		Extra:     make(map[string]string),
		MindmapID: mindMapID,
	}
}
