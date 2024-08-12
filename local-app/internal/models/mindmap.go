package models

type Mindmap struct {
	ID       int    `json:"-"`
	Name     string `json:"name"`
	Owner    string `json:"-"`
	IsPublic bool   `json:"-"`
	Root     *Node  `json:"root"`
}

type ExportableMindmap struct {
	Name string `json:"name"`
	Root *Node  `json:"root"`
}

func NewMindmap(id int, name string, owner string, isPublic bool) *Mindmap {
	return &Mindmap{
		ID:       id,
		Name:     name,
		Owner:    owner,
		IsPublic: isPublic,
	}
}

func (m *Mindmap) ToExportable() *ExportableMindmap {
	return &ExportableMindmap{
		Name: m.Name,
		Root: m.Root,
	}
}
