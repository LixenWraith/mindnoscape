package model

// Command represents a user command with its scope, operation, and arguments
type Command struct {
	Scope     string
	Operation string
	Args      []string
}
