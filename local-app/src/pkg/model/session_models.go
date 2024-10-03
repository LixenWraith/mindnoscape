package model

import "time"

type Session struct {
	ID           string
	User         *User
	Mindmap      *Mindmap
	LastActivity time.Time
}
