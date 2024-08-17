package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
	"unicode/utf8"
)

var ErrInterrupted = errors.New("interrupted")

type UI struct {
	writer   io.Writer
	useColor bool
	//	SystemUI *SystemUI
	UserUI    *UserUI
	MindmapUI *MindmapUI
	NodeUI    *NodeUI
}

func NewUI(w io.Writer, useColor bool) *UI {
	return &UI{
		writer:   w,
		useColor: useColor,
		//		SystemUI: NewSystemUI(w, useColor),
		UserUI:    NewUserUI(w, useColor),
		MindmapUI: NewMindmapUI(w, useColor),
		NodeUI:    NewNodeUI(w, useColor),
	}
}

func (u *UI) colorize(message string, color Color) string {
	if !u.useColor || color == ColorDefault {
		return message
	}
	return fmt.Sprintf("%s%s%s", color, message, ColorDefault)
}

func (u *UI) Print(message string) {
	fmt.Fprint(u.writer, message)
}

func (u *UI) Printf(format string, args ...interface{}) {
	fmt.Fprintf(u.writer, format, args...)
}

func (u *UI) Println(message string) {
	fmt.Fprintln(u.writer, message)
}

func (u *UI) PrintColored(message string, color Color) {
	fmt.Fprint(u.writer, u.colorize(message, color))
}

func (u *UI) PrintlnColored(message string, color Color) {
	fmt.Fprintln(u.writer, u.colorize(message, color))
}

func (u *UI) Error(message string) {
	u.Printf(fmt.Sprintf("%s!%s %s\n", ColorRed, ColorLightOrange, message))
}

func (u *UI) Success(message string) {
	u.PrintlnColored(message, ColorLightGreen)
}

func (u *UI) Warning(message string) {
	u.Printf(fmt.Sprintf("%s?%s %s\n", ColorLightRed, ColorLightYellow, message))
}

func (u *UI) Info(message string) {
	u.PrintlnColored(message, ColorGray)
}

func (u *UI) GetPromptString(user, mindmap string) string {
	var promptBuilder strings.Builder
	if user != "" {
		promptBuilder.WriteString(u.colorize(user, ColorLightBlue))
		if mindmap != "" {
			promptBuilder.WriteString(u.colorize(" @ ", ColorWhite))
			promptBuilder.WriteString(u.colorize(mindmap, ColorLightPurple))
		}
		promptBuilder.WriteString(" ")
	}
	promptBuilder.WriteString(u.colorize("> ", ColorGreen))
	return promptBuilder.String()
}

func (u *UI) ReadLine(prompt string) (string, error) {
	u.Print(prompt)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	var input []rune
	var position int
	buffer := make([]byte, 1)
	promptLen := u.visibleLength(prompt)

	redrawLine := func() {
		u.Print("\r\x1b[K")                                       // Move to start of line and clear the entire line
		u.Print(prompt)                                           // Print prompt
		u.Print(string(input))                                    // Print current input
		u.Print("\r")                                             // Move cursor back to start
		u.Print("\x1b[" + fmt.Sprintf("%dC", promptLen+position)) // Move cursor to correct position
	}

	for {
		_, err := os.Stdin.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to read input: %w", err)
		}

		switch buffer[0] {
		case 13, 10: // Enter key (CR or LF)
			u.Println("\r")
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
		default:
			r, _ := utf8.DecodeRune(buffer)
			input = append(input[:position], append([]rune{r}, input[position:]...)...)
			position++
			redrawLine()
		}
	}

	return string(input), nil
}

func (u *UI) ReadPassword(prompt string) (string, error) {
	u.Print(prompt)

	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	u.Println("") // Print a newline after the password input
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	return string(password), nil
}

func (u *UI) readEscapeSequence() (string, error) {
	buf := make([]byte, 2)
	_, err := os.Stdin.Read(buf)
	if err != nil {
		return "", fmt.Errorf("failed to read escape sequence: %w", err)
	}
	return string(buf[1]), nil
}

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
