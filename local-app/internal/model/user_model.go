// Package model defines the data structures used throughout the Mindnoscape application.
package model

import "time"

// User represents a user account in the Mindnoscape application.
type User struct {
	ID           int              `json:"id" xml:"id,attr"`
	Username     string           `json:"username" xml:"username"`
	PasswordHash []byte           `json:"-" xml:"-"`
	Mindmaps     map[int]*Mindmap `json:"mindmaps,omitempty" xml:"mindmaps>mindmaps,omitempty"`
	Active       bool             `json:"active" xml:"active,attr"`
	Created      time.Time        `json:"created" xml:"created,attr"`
	Updated      time.Time        `json:"updated" xml:"updated,attr"`
}

// UserInfo contains basic information about a user.
type UserInfo struct {
	ID           int
	Username     string
	PasswordHash []byte
	Active       bool
	MindmapCount *int
}

// UserFilter defines the options for filtering users.
type UserFilter struct {
	ID           bool
	Username     bool
	PasswordHash bool
	Active       bool
}
