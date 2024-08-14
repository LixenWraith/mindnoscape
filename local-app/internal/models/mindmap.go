package models

type Mindmap struct {
	ID       int           `json:"-" xml:"-"`
	Name     string        `json:"name" xml:"name,attr"`
	Owner    string        `json:"-" xml:"-"`
	IsPublic bool          `json:"-" xml:"-"`
	Root     *Node         `json:"root" xml:"root"`
	Nodes    map[int]*Node `json:"-" xml:"-"`
	MaxIndex int           `json:"-" xml:"-"`
}

type ExportableMindmap struct {
	Name string `json:"name" xml:"name,attr"`
	Root *Node  `json:"root" xml:"root"`
}

func NewMindmap(id int, name string, owner string, isPublic bool) *Mindmap {
	return &Mindmap{
		ID:       id,
		Name:     name,
		Owner:    owner,
		IsPublic: isPublic,
		Root:     &Node{Index: 0, ParentID: -1, Content: name, LogicalIndex: "0"},
		Nodes:    make(map[int]*Node),
		MaxIndex: 0,
	}
}

func (m *Mindmap) ToExportable() *ExportableMindmap {
	return &ExportableMindmap{
		Name: m.Name,
		Root: m.Root,
	}
}
