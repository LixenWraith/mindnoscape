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

func (n *Node) AddChild(child *Node) {
    n.Children = append(n.Children, child)
}

func (n *Node) RemoveChild(index int) {
    if index < 0 || index >= len(n.Children) {
        return
    }
    n.Children = append(n.Children[:index], n.Children[index+1:]...)
}

func (n *Node) SetExtra(key, value string) {
    n.Extra[key] = value
}

func (n *Node) GetExtra(key string) (string, bool) {
    value, exists := n.Extra[key]
    return value, exists
}

func (n *Node) RemoveExtra(key string) {
    delete(n.Extra, key)
}

