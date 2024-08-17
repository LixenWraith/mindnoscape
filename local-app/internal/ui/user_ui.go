package ui

import (
	"io"

	"mindnoscape/local-app/internal/models"
)

type UserUI struct {
	visualizer *Visualizer
}

func NewUserUI(w io.Writer, useColor bool) *UserUI {
	return &UserUI{
		visualizer: NewVisualizer(w, useColor),
	}
}

// UserInfo displays information about a single user
func (uui *UserUI) UserInfo(user *models.User) {
	uui.visualizer.Printf("Username: %s\n", user.Username)
	// Add more user information as needed
}

// UserList displays a list of users
func (uui *UserUI) UserList(users []*models.User) {
	if len(users) == 0 {
		uui.visualizer.Println("No users found.")
		return
	}

	uui.visualizer.Println("User List:")
	for _, user := range users {
		uui.visualizer.Printf("- %s\n", user.Username)
	}
}

// UserCreated displays a message when a user is successfully created
func (uui *UserUI) UserCreated(username string) {
	uui.visualizer.PrintColored(
		"User created successfully: "+username+"\n",
		ColorGreen,
	)
}

// UserDeleted displays a message when a user is successfully deleted
func (uui *UserUI) UserDeleted(username string) {
	uui.visualizer.PrintColored(
		"User deleted successfully: "+username+"\n",
		ColorYellow,
	)
}

// UserModified displays a message when a user is successfully modified
func (uui *UserUI) UserModified(username string) {
	uui.visualizer.PrintColored(
		"User modified successfully: "+username+"\n",
		ColorBlue,
	)
}
