package models

type MindMap struct {
	ID       int    `json:"-"`
	Name     string `json:"name"`
	Owner    string `json:"-"`
	IsPublic bool   `json:"-"`
	Root     *Node  `json:"root"`
}

type ExportableMindMap struct {
	Name string `json:"name"`
	Root *Node  `json:"root"`
}

func NewMindMap(id int, name string, owner string, isPublic bool) *MindMap {
	return &MindMap{
		ID:       id,
		Name:     name,
		Owner:    owner,
		IsPublic: isPublic,
	}
}

func (m *MindMap) ToExportable() *ExportableMindMap {
	return &ExportableMindMap{
		Name: m.Name,
		Root: m.Root,
	}
}
