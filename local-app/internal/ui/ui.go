// Package ui provides user interface functionality for the Mindnoscape application.
// This file contains the main UI structure and methods for handling user input and output.
package ui

import (
	"errors"
	"fmt"
	"golang.org/x/term"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

// ErrInterrupted is returned when user input is interrupted (e.g., by Ctrl+C).
var ErrInterrupted = errors.New("interrupted")

// UI represents the user interface of the application.
type UI struct {
	writer     io.Writer
	useColor   bool
	visualizer *Visualizer
	UserUI     *UserUI
	MindmapUI  *MindmapUI
	NodeUI     *NodeUI
}

// NewUI creates a new UI instance with the specified writer and color setting.
func NewUI(w io.Writer, useColor bool) *UI {
	return &UI{
		writer:     w,
		useColor:   useColor,
		visualizer: NewVisualizer(w, useColor),
		//		SystemUI: NewSystemUI(w, useColor),
		UserUI:    NewUserUI(w, useColor),
		MindmapUI: NewMindmapUI(w, useColor),
		NodeUI:    NewNodeUI(w, useColor),
	}
}

// Message displays a standard message.
func (u *UI) Message(args ...any) {
	if len(args) == 1 {
		// If only one argument is provided, use Println
		u.visualizer.PrintColored(fmt.Sprintf("%v\n", args[0]), ColorWhite)
	} else if len(args) > 1 {
		// If more than one argument is provided, treat it like Printf
		format, ok := args[0].(string)
		if ok {
			u.visualizer.Printf(format, args[1:]...)
		}
	}
}

// Error displays an error message.
func (u *UI) Error(message string) {
	u.visualizer.Printf(fmt.Sprintf("%s!%s %s\n", ColorRed, ColorLightOrange, message))
}

// Success displays a success message.
func (u *UI) Success(message string) {
	u.visualizer.PrintColored(message+"\n", ColorLightGreen)
}

// Warning displays a warning message.
func (u *UI) Warning(message string) {
	u.visualizer.Printf(fmt.Sprintf("%s?%s %s\n", ColorLightRed, ColorLightYellow, message))
}

// Info displays an informational message.
func (u *UI) Info(message string) {
	u.visualizer.PrintColored(message+"\n", ColorGray)
}

// Prompt generates a prompt string based on the current user and mindmap and displays the prompt.
func (u *UI) Prompt(username, mindmap string) {
	u.visualizer.Print(u.GetPromptString(username, mindmap))
}

func (u *UI) GetPromptString(username, mindmap string) string {
	var promptBuilder strings.Builder

	// Add current time
	currentTime := time.Now().Format("[15:04:05] ")
	promptBuilder.WriteString(u.colorize(currentTime, ColorGray))

	if mindmap != "" {
		promptBuilder.WriteString(u.colorize(mindmap, ColorLightBlue))
		promptBuilder.WriteString(u.colorize(" @ ", ColorWhite))
	}

	if username != "" {
		promptBuilder.WriteString(u.colorize(username, ColorLightPurple))
		promptBuilder.WriteString(" ")
	}

	promptBuilder.WriteString(u.colorize("> ", ColorGreen))
	return promptBuilder.String()
}

// ReadLine reads a line of input from the user with the given prompt.
func (u *UI) ReadLine(prompt string) (string, error) {
	// Display the prompt
	u.visualizer.Print(prompt)

	// Set the terminal to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Initialize variables for input handling
	var input []rune
	var position int
	buffer := make([]byte, 1)
	promptLen := u.visibleLength(prompt)

	// Function to redraw the current input line
	redrawLine := func() {
		u.visualizer.Print("\r\x1b[K")                                       // Move to start of line and clear the entire line
		u.visualizer.Print(prompt)                                           // Print prompt
		u.visualizer.Print(string(input))                                    // Print current input
		u.visualizer.Print("\r")                                             // Move cursor back to start
		u.visualizer.Print("\x1b[" + fmt.Sprintf("%dC", promptLen+position)) // Move cursor to correct position
	}

	// Main input loop
	for {
		_, err := os.Stdin.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to read input: %w", err)
		}

		// Handle different input cases (Enter, Ctrl+C, Backspace, arrow keys, etc.)
		switch buffer[0] {
		case 13, 10: // Enter key (CR or LF)
			u.visualizer.Println("\r")
			return string(input), nil
		case 3: // Ctrl+C
			return "", ErrInterrupted
		case 127, 8: // Backspace
			if position > 0 {
				input = append(input[:position-1], input[position:]...)
				position--
				redrawLine()
			}
		case 27: // Escape sequence (e.g., arrow keys)
			escSeq, err := u.readEscapeSequence()
			if err != nil {
				return "", err
			}
			switch escSeq {
			case "C": // Right arrow
				if position < len(input) {
					position++
					redrawLine()
				}
			case "D": // Left arrow
				if position > 0 {
					position--
					redrawLine()
				}
			}
		default: // Regular characters
			r, _ := utf8.DecodeRune(buffer)
			input = append(input[:position], append([]rune{r}, input[position:]...)...)
			position++
			redrawLine()
		}
	}

	return string(input), nil
}

// ReadPassword reads a password from the user, hiding the input.
func (u *UI) ReadPassword(prompt string) (string, error) {
	u.visualizer.Print(prompt)

	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	u.visualizer.Println("") // Print a newline after the password input
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	return string(password), nil
}

// readEscapeSequence reads an escape sequence from stdin.
func (u *UI) readEscapeSequence() (string, error) {
	buf := make([]byte, 2)
	_, err := os.Stdin.Read(buf)
	if err != nil {
		return "", fmt.Errorf("failed to read escape sequence: %w", err)
	}
	return string(buf[1]), nil
}

// visibleLength calculates the visible length of a string, ignoring ANSI escape sequences.
func (u *UI) visibleLength(s string) int {
	visible := 0
	inEscapeSeq := false
	for _, r := range s {
		if inEscapeSeq {
			if r == 'm' {
				inEscapeSeq = false
			}
		} else if r == '\x1b' {
			inEscapeSeq = true
		} else {
			visible++
		}
	}
	return visible
}

// colorize applies color to a message if color is enabled.
func (u *UI) colorize(message string, color Color) string {
	if !u.useColor || color == ColorDefault {
		return message
	}
	return fmt.Sprintf("%s%s%s", color, message, ColorDefault)
}
