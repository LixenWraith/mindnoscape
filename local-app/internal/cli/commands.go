package cli

import (
	"fmt"
	"io"
	"mindnoscape/local-app/internal/ui"
	"sort"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"mindnoscape/local-app/internal/mindmap"
	"mindnoscape/local-app/internal/models"
	"mindnoscape/local-app/internal/storage"
)

func (c *CLI) handleUser(args []string) error {
	if len(args) == 0 {
		fmt.Printf("Current user: %s\n", c.CurrentUser)
		return nil
	}

	switch args[0] {
	case "--new":
		return c.handleNewUser(args[1:])
	case "--mod":
		return c.handleModifyUser(args[1:])
	case "--del":
		return c.handleDeleteUser(args[1:])
	default:
		return c.handleChangeUser(args)
	}
}

func (c *CLI) handleNewUser(args []string) error {
	var username, password string
	var err error

	if len(args) > 0 {
		username = args[0]
	} else {
		username, err = c.promptForInput("Enter new username: ")
		if err != nil {
			return err
		}
	}

	// Check if user already exists
	exists, err := c.Mindmap.Store.UserExists(username)
	if err != nil {
		return fmt.Errorf("error checking user existence: %v", err)
	}
	if exists {
		return fmt.Errorf("user '%s' already exists", username)
	}

	if len(args) > 1 {
		password = args[1]
	} else {
		password, err = c.promptForPassword("Enter password for new user: ")
		if err != nil {
			return err
		}
	}

	err = c.Mindmap.Store.AddUser(username, password)
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	fmt.Printf("User '%s' created successfully\n", username)
	return nil
}

func (c *CLI) handleDeleteUser(args []string) error {
	var username, password string
	var err error

	switch len(args) {
	case 0:
		username, err = c.promptForInput("Enter username to delete: ")
		if err != nil {
			return err
		}
		password, err = c.promptForPassword("Enter password: ")
		if err != nil {
			return err
		}
	case 1:
		username = args[0]
		password, err = c.promptForPassword("Enter password: ")
		if err != nil {
			return err
		}
	case 2:
		username = args[0]
		password = args[1]
	default:
		return fmt.Errorf("usage: user --del [<username> [password]]")
	}

	// Authenticate user
	authenticated, err := c.Mindmap.Store.AuthenticateUser(username, password)
	if err != nil {
		return fmt.Errorf("authentication error: %v", err)
	}
	if !authenticated {
		return fmt.Errorf("invalid username or password")
	}

	// Delete user and their mindmaps
	err = c.Mindmap.Store.DeleteUser(username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	// If the deleted user was the current user, switch to guest
	if c.CurrentUser == username {
		c.CurrentUser = "guest"
		_ = c.Mindmap.ChangeUser("guest")
	}

	c.UI.Success(fmt.Sprintf("User '%s' and all associated mindmaps deleted successfully", username))
	return nil
}

func (c *CLI) handleModifyUser(args []string) error {
	var username, currentPassword, newUsername, newPassword string
	var err error

	if c.CurrentUser != "guest" {
		username = c.CurrentUser
	} else if len(args) > 0 {
		username = args[0]
	} else {
		username, err = c.promptForInput("Enter username to modify: ")
		if err != nil {
			return err
		}
	}

	currentPassword, err = c.promptForPassword("Enter current password: ")
	if err != nil {
		return err
	}

	// Authenticate user
	authenticated, err := c.Mindmap.Store.AuthenticateUser(username, currentPassword)
	if err != nil {
		return fmt.Errorf("authentication error: %v", err)
	}
	if !authenticated {
		return fmt.Errorf("invalid username or password")
	}

	newUsername, err = c.promptForInput("Enter new username (leave empty to keep current): ")
	if err != nil {
		return err
	}

	newPassword, err = c.promptForPassword("Enter new password (leave empty to keep current): ")
	if err != nil {
		return err
	}

	err = c.Mindmap.Store.ModifyUser(username, newUsername, newPassword)
	if err != nil {
		return fmt.Errorf("failed to update user: %v", err)
	}

	fmt.Println("User updated successfully")
	if newUsername != "" && newUsername != username {
		c.CurrentUser = newUsername
	}
	return nil
}

func (c *CLI) handleChangeUser(args []string) error {
	var username, password string
	var err error

	if len(args) > 0 {
		username = args[0]
	} else { // Never happens
		username, err = c.promptForInput("Enter username: ")
		if err != nil {
			return err
		}
	}

	if len(args) > 1 {
		password = args[1]
	} else {
		if username == "guest" {
			password = ""
		} else {
			password, err = c.promptForPassword("Enter password: ")
			if err != nil {
				return err
			}
		}
	}

	authenticated, err := c.Mindmap.Store.AuthenticateUser(username, password)
	if err != nil {
		return fmt.Errorf("authentication error: %v", err)
	}
	if !authenticated {
		return fmt.Errorf("invalid username or password")
	}

	err = c.Mindmap.ChangeUser(username)
	if err != nil {
		return fmt.Errorf("failed to switch user: %v", err)
	}

	c.CurrentUser = username
	fmt.Printf("Switched to user: %s\n", username)
	c.UpdatePrompt()
	return nil
}

func (c *CLI) promptForInput(prompt string) (string, error) {
	fmt.Print(prompt)
	input, err := c.RL.Readline()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func (c *CLI) promptForPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	passwordBytes, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println() // Print a newline after the password input
	return string(passwordBytes), nil
}

func (c *CLI) handleModifyAccess(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: access <mindmap name> <public|private>")
	}

	mindmapName := args[0]
	access := args[1]

	isPublic := false
	if access == "public" {
		isPublic = true
	} else if access != "private" {
		return fmt.Errorf("invalid access option: use 'public' or 'private'")
	}

	err := c.Mindmap.Store.ModifyMindmapAccess(mindmapName, c.CurrentUser, isPublic)
	if err != nil {
		return fmt.Errorf("failed to update mindmap access: %v", err)
	}

	fmt.Printf("Mindmap '%s' access set to %s\n", mindmapName, access)
	return nil
}

func (c *CLI) handleNewNode(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: new <mindmap name>")
	}

	name := args[0]
	isPublic := c.CurrentUser == "guest"
	err := c.Mindmap.AddMindmap(name, isPublic)
	if err != nil {
		return err
	}

	c.UI.Success(fmt.Sprintf("New mindmap '%s' created and switched to", name))
	return nil
}

func (c *CLI) handleChangeMindmap(args []string) error {
	mindmaps, err := c.Mindmap.ListMindmap()
	if err != nil {
		return err
	}
	if len(mindmaps) == 0 {
		fmt.Println("No mindmaps available, use 'new' to create a new mindmap or 'load' to load one from a file")
		return nil
	}

	if len(args) == 0 {
		if c.Mindmap.CurrentMindmap == nil {
			c.UI.Info("Not currently in any mindmap, use 'switch <mindmap name>' to switch to a mindmap")
			return nil
		}
		c.Mindmap.CurrentMindmap = nil
		c.UI.Info("Switched out of the current mindmap")
		return nil
	}

	name := args[0]
	err = c.Mindmap.ChangeMindmap(name)
	if err != nil {
		return err
	}

	c.UI.Success(fmt.Sprintf("Switched to mindmap '%s'", name))
	c.UpdatePrompt()
	return nil
}

func (c *CLI) handleListMindmap(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("usage: list")
	}

	mindmaps, err := c.Mindmap.Store.GetAllMindmaps(c.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to retrieve mindmaps: %v", err)
	}

	if len(mindmaps) == 0 {
		c.UI.Println("No mindmaps available")
	} else {
		c.UI.Println("Available mindmaps:")
		for _, mm := range mindmaps {
			accessSymbol := "+"
			accessColor := ui.ColorGreen
			if !mm.IsPublic {
				accessSymbol = "-"
				accessColor = ui.ColorRed
			}
			c.UI.Print(mm.Name + " ")
			c.UI.PrintColored(accessSymbol, accessColor)
			if mm.Owner != c.CurrentUser {
				c.UI.Printf(" (owner: %s)", mm.Owner)
			}
			c.UI.Println("")
		}
	}

	return nil
}

func (c *CLI) handleAddNode(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: add <parent> <content> [<extra field label>:<extra field value>]... [--index]")
	}

	parentIdentifier := args[0]
	content := args[1]
	extra := make(map[string]string)
	useIndex := false

	// Process extra fields and check for --index flag
	for _, arg := range args[2:] {
		if arg == "--index" {
			useIndex = true
		} else {
			parts := strings.SplitN(arg, ":", 2)
			if len(parts) == 2 {
				extra[parts[0]] = parts[1]
			}
		}
	}

	err := c.Mindmap.AddNode(parentIdentifier, content, extra, useIndex)
	if err != nil {
		return err
	}

	c.UI.Success("Node added successfully")
	return nil
}

func (c *CLI) handleDeleteNode(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: del <node> [--index]")
	}

	identifier := args[0]
	useIndex := false

	// Check for --index flag
	if len(args) > 1 && args[1] == "--index" {
		useIndex = true
	}

	err := c.Mindmap.DeleteNode(identifier, useIndex)
	if err != nil {
		return err
	}

	c.UI.Success("Node deleted successfully")
	return nil
}

func (c *CLI) handleDeleteMindmap(args []string) error {
	if len(args) == 0 {
		// Clear all mindmaps owned by the current user
		mindmaps, err := c.Mindmap.ListMindmap()
		if err != nil {
			return fmt.Errorf("failed to get mindmaps: %v", err)
		}

		clearedCount := 0
		for _, mm := range mindmaps {
			if mm.Owner == c.CurrentUser {
				err := c.Mindmap.DeleteMindmap(mm.Name)
				if err != nil {
					return fmt.Errorf("failed to delete mindmap '%s': %v", mm.Name, err)
				}
				clearedCount++
			}
		}

		c.UI.Success(fmt.Sprintf("%d mind map(s) deleted", clearedCount))
	} else {
		// Delete a specific mindmap
		mindmapName := args[0]
		err := c.Mindmap.DeleteMindmap(mindmapName)
		if err != nil {
			return fmt.Errorf("failed to delete mindmap '%s': %v", mindmapName, err)
		}

		c.UI.Success(fmt.Sprintf("Mind map '%s' deleted", mindmapName))
	}

	c.UpdatePrompt()
	return nil
}

func (c *CLI) handleModifyNode(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: mod <node> <content> [<extra field label>:<extra field value>]... [--index]")
	}

	identifier := args[0]
	content := args[1]
	extra := make(map[string]string)
	useIndex := false

	// Process extra fields and check for --index flag
	for i := 2; i < len(args); i++ {
		if args[i] == "--index" {
			useIndex = true
		} else {
			parts := strings.SplitN(args[i], ":", 2)
			if len(parts) == 2 {
				extra[parts[0]] = parts[1]
			}
		}
	}

	err := c.Mindmap.ModifyNode(identifier, content, extra, useIndex)
	if err != nil {
		return err
	}

	c.UI.Success("Node modified successfully")
	return nil
}

func (c *CLI) handleMoveNode(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: move <source> <target> [--index]")
	}

	sourceIdentifier := args[0]
	targetIdentifier := args[1]
	useIndex := false

	// Check for --index flag
	if len(args) > 2 && args[2] == "--index" {
		useIndex = true
	}

	err := c.Mindmap.MoveNode(sourceIdentifier, targetIdentifier, useIndex)
	if err != nil {
		return err
	}

	c.UI.Success("Node moved successfully")
	return nil
}

func (c *CLI) handleInsertNode(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: insert <source> <target> [--index]")
	}

	sourceIdentifier := args[0]
	targetIdentifier := args[1]
	useIndex := false

	// Check for --index flag
	if len(args) > 2 && args[2] == "--index" {
		useIndex = true
	}

	err := c.Mindmap.InsertNode(sourceIdentifier, targetIdentifier, useIndex)
	if err != nil {
		return err
	}

	c.UI.Success("Node inserted successfully")
	return nil
}

func (c *CLI) handleShowMindmap(args []string) error {
	logicalIndex := ""
	showIndex := false

	for _, arg := range args {
		if arg == "--index" {
			showIndex = true
		} else {
			logicalIndex = arg
		}
	}

	output, err := c.Mindmap.ShowMindmap(logicalIndex, showIndex)
	if err != nil {
		return err
	}

	for _, line := range output {
		c.printColoredLine(line)
	}

	return nil
}

func (c *CLI) printColoredLine(line string) {
	colorMap := map[string]ui.Color{
		"{{yellow}}":    ui.ColorYellow,
		"{{orange}}":    ui.ColorOrange,
		"{{darkbrown}}": ui.ColorDarkBrown,
		"{{default}}":   ui.ColorDefault,
	}

	for len(line) > 0 {
		startIndex := strings.Index(line, "{{")
		if startIndex == -1 {
			c.UI.Print(line)
			break
		}

		endIndex := strings.Index(line, "}}")
		if endIndex == -1 {
			c.UI.Print(line)
			break
		}

		// Print the part before the color code
		if startIndex > 0 {
			c.UI.Print(line[:startIndex])
		}

		colorCode := line[startIndex : endIndex+2]
		color, exists := colorMap[colorCode]
		if !exists {
			color = ui.ColorDefault
		}

		// Find the next color code or the end of the string
		nextStartIndex := strings.Index(line[endIndex+2:], "{{")
		if nextStartIndex == -1 {
			// No more color codes, print the rest of the line
			c.UI.PrintColored(line[endIndex+2:], color)
			break
		} else {
			// Print the part until the next color code
			c.UI.PrintColored(line[endIndex+2:endIndex+2+nextStartIndex], color)
			line = line[endIndex+2+nextStartIndex:]
		}
	}
	c.UI.Println("") // New line at the end
}

func (c *CLI) handleSortNode(args []string) error {
	identifier := ""
	field := ""
	reverse := false
	useIndex := false
	for i := 0; i < len(args); i++ {
		arg := strings.ToLower(args[i])
		switch arg {
		case "--reverse":
			reverse = true
		case "--index":
			useIndex = true
		default:
			if identifier == "" {
				identifier = args[i]
			} else if field == "" {
				field = args[i]
			}
		}
	}
	err := c.Mindmap.Sort(identifier, field, reverse, useIndex)
	if err != nil {
		return err
	}
	c.UI.Success("Sorted successfully")
	return nil
}

func (c *CLI) handleSaveMindmap(args []string) error {
	filename := "mindmap.json"
	format := "json"

	if len(args) >= 1 {
		filename = args[0]
	}
	if len(args) >= 2 {
		format = args[1]
	}

	if c.Mindmap.CurrentMindmap == nil {
		return fmt.Errorf("no mindmap selected")
	}

	err := storage.SaveToFile(c.Mindmap.Store, c.Mindmap.CurrentMindmap.Root.Content, c.CurrentUser, filename, format)
	if err != nil {
		return err
	}

	fmt.Printf("Mind map saved to %s in %s format\n", filename, format)
	return nil
}

func (c *CLI) handleLoadMindmap(args []string) error {
	filename := "mindmap.json"
	format := "json"

	if len(args) >= 1 {
		filename = args[0]
	}
	if len(args) >= 2 {
		format = args[1]
	}

	// Import the file into a temporary root node
	tempRoot, err := storage.ImportFromFile(filename, format)
	if err != nil {
		return err
	}

	mindmapName := tempRoot.Content
	exists, err := c.Mindmap.Store.MindmapExists(mindmapName, c.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to check if mindmap exists: %v", err)
	}

	isCurrentMindmap := c.Mindmap.CurrentMindmap != nil && c.Mindmap.CurrentMindmap.Root.Content == mindmapName

	// If we're currently in the mindmap we're loading, switch out first
	if isCurrentMindmap {
		c.Mindmap.CurrentMindmap = nil
		c.Prompt = "> "
		fmt.Printf("Switched out of mindmap '%s' before reloading.\n", mindmapName)
	}

	// Check if the user has permission to modify the mindmap
	hasPermission, err := c.Mindmap.Store.HasMindmapPermission(mindmapName, c.CurrentUser)
	if err != nil {
		return fmt.Errorf("failed to check mindmap permissions: %v", err)
	}
	if !hasPermission {
		return fmt.Errorf("user %s does not have permission to modify mindmap '%s'", c.CurrentUser, mindmapName)
	}

	if exists {
		fmt.Printf("Replacing content of existing mindmap '%s'\n", mindmapName)
		// Delete all nodes except the root
		nodes, err := c.Mindmap.Store.GetNode(mindmapName, c.CurrentUser, -1)
		if err != nil {
			return fmt.Errorf("failed to retrieve nodes for mindmap '%s': %v", mindmapName, err)
		}
		for _, node := range nodes {
			if node.ParentID != -1 { // Skip the root node
				err = c.Mindmap.Store.DeleteNode(mindmapName, c.CurrentUser, node.Index)
				if err != nil {
					return fmt.Errorf("failed to delete node %d from mindmap '%s': %v", node.Index, mindmapName, err)
				}
			}
		}
		// Remove from in-memory map as well
		delete(c.Mindmap.Mindmaps, mindmapName)
	} else {
		fmt.Printf("Creating new mindmap '%s' from file\n", mindmapName)
	}

	// Load the content into the mindmap
	err = storage.LoadFromFile(c.Mindmap.Store, mindmapName, c.CurrentUser, filename, format)
	if err != nil {
		return fmt.Errorf("failed to load nodes for mindmap '%s': %v", mindmapName, err)
	}

	// Create a new Mindmap struct
	newMindmap := &mindmap.Mindmap{
		Nodes: make(map[int]*models.Node),
	}
	c.Mindmap.Mindmaps[mindmapName] = newMindmap

	// Reload the mind map from storage
	err = c.Mindmap.LoadNodes(mindmapName)
	if err != nil {
		return fmt.Errorf("failed to reload mind map after load: %v", err)
	}

	fmt.Printf("Mind map '%s' loaded from %s\n", mindmapName, filename)

	// If we were in the mindmap before, switch back to it
	if isCurrentMindmap {
		err = c.Mindmap.ChangeMindmap(mindmapName)
		if err != nil {
			return fmt.Errorf("failed to switch back to mindmap '%s': %v", mindmapName, err)
		}
		c.Prompt = fmt.Sprintf("%s > ", mindmapName)
		fmt.Printf("Switched back to mindmap '%s'\n", mindmapName)
	}

	return nil
}

func (c *CLI) handleFindNode(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: find <query> [--index]")
	}

	query := args[0]
	showIndex := false

	// Check for --index flag
	if len(args) > 1 && args[1] == "--index" {
		showIndex = true
	}

	// If the query is enclosed in quotes, remove them
	if strings.HasPrefix(query, "\"") && strings.HasSuffix(query, "\"") {
		query = query[1 : len(query)-1]
	}

	output, err := c.Mindmap.FindNode(query, showIndex)
	if err != nil {
		return fmt.Errorf("failed to find nodes: %v", err)
	}

	for _, line := range output {
		c.UI.Println(line)
	}

	return nil
}

func (c *CLI) printNode(node *models.Node, showIndex bool) {
	fmt.Printf("LogicalIndex: %s, Content: %s", node.LogicalIndex, node.Content)
	if showIndex {
		fmt.Printf(" [%d]", node.Index)
	}

	// Add extra fields
	if len(node.Extra) > 0 {
		var extraFields []string
		for k, v := range node.Extra {
			extraFields = append(extraFields, fmt.Sprintf("%s:%s", k, v))
		}
		sort.Strings(extraFields) // Sort extra fields for consistent output
		fmt.Printf(" %s", strings.Join(extraFields, " "))
	}

	fmt.Println() // End the line
}

func (c *CLI) handleUndo(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: undo")
	}

	err := c.Mindmap.Undo()
	if err != nil {
		return err
	}

	c.UI.Success("Undo successful")
	return nil
}

func (c *CLI) handleRedo(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: redo")
	}

	err := c.Mindmap.Redo()
	if err != nil {
		return err
	}

	c.UI.Success("Redo successful")
	return nil
}

func (c *CLI) handleHelp(args []string) error {
	if len(args) > 0 {
		c.printHelp(args[0])
	} else {
		c.printHelp("")
	}
	return nil
}

func (c *CLI) handleExit() error {
	fmt.Println("Exiting...")
	err := c.RL.Close()
	if err != nil {
		fmt.Printf("Error closing readline: %v\n", err)
	}
	return fmt.Errorf("exit requested: %w", io.EOF)
}
