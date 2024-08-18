// Package models defines the data structures used throughout the Mindnoscape application.
package models

// User represents a user account in the Mindnoscape application.
type User struct {
	ID           int
	Username     string `json:"username"`
	PasswordHash []byte `json:"-"` // The "-" tag ensures this field is not included in JSON output
}

// UserInfo contains basic information about a user.
type UserInfo struct {
	ID       int // TODO: Get this from db for user list
	Username string
}

// NewUser creates a new User instance with the given username.
func NewUser(username string) *User {
	return &User{
		Username: username,
	}
}
