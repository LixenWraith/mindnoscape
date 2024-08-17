package models

type Mindmap struct {
	ID       int           `json:"-" xml:"-"`
	Name     string        `json:"name" xml:"name,attr"`
	Owner    string        `json:"-" xml:"-"`
	IsPublic bool          `json:"-" xml:"-"`
	Root     *Node         `json:"root" xml:"root"`
	Nodes    map[int]*Node `json:"-" xml:"-"`
	MaxID    int           `json:"-" xml:"-"`
}

type MindmapInfo struct {
	ID       int
	Name     string
	IsPublic bool
	Owner    string
}

func NewMindmap(id int, name string, owner string, isPublic bool) *Mindmap {
	return &Mindmap{
		ID:       id,
		Name:     name,
		Owner:    owner,
		IsPublic: isPublic,
		Root:     &Node{ID: 0, ParentID: -1, Content: name, Index: "0"},
		Nodes:    make(map[int]*Node),
		MaxID:    0,
	}
}
