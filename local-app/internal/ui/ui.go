package ui

import (
	"fmt"
	"io"
	"strings"
)

type Color string

const (
	ColorDefault     Color = "\033[0m"
	ColorBlack       Color = "\033[38;2;0;0;0m"
	ColorDarkGray    Color = "\033[38;2;100;100;100m"
	ColorGray        Color = "\033[38;2;150;150;150m"
	ColorWhite       Color = "\033[38;2;255;255;255m"
	ColorBrightWhite Color = "\033[38;2;255;255;255;1m"

	ColorLightRed Color = "\033[38;2;255;150;150m"
	ColorRed      Color = "\033[38;2;255;0;0m"
	ColorDarkRed  Color = "\033[38;2;150;0;0m"

	ColorLightGreen Color = "\033[38;2;150;255;150m"
	ColorGreen      Color = "\033[38;2;0;255;0m"
	ColorDarkGreen  Color = "\033[38;2;0;150;0m"

	ColorLightYellow Color = "\033[38;2;255;255;150m"
	ColorYellow      Color = "\033[38;2;255;255;0m"
	ColorDarkYellow  Color = "\033[38;2;150;150;0m"

	ColorLightBlue Color = "\033[38;2;150;150;255m"
	ColorBlue      Color = "\033[38;2;0;0;255m"
	ColorDarkBlue  Color = "\033[38;2;0;0;150m"

	ColorLightBrown Color = "\033[38;2;210;180;140m"
	ColorBrown      Color = "\033[38;2;165;42;42m"
	ColorDarkBrown  Color = "\033[38;2;101;67;33m"

	ColorLightPurple Color = "\033[38;2;200;150;255m"
	ColorPurple      Color = "\033[38;2;128;0;128m"
	ColorDarkPurple  Color = "\033[38;2;75;0;130m"

	ColorLightOrange Color = "\033[38;2;255;200;150m"
	ColorOrange      Color = "\033[38;2;255;165;0m"
	ColorDarkOrange  Color = "\033[38;2;255;140;0m"

	ColorPink Color = "\033[38;2;255;192;203m"
)

type UI struct {
	writer   io.Writer
	useColor bool
}

func NewUI(w io.Writer, useColor bool) *UI {
	return &UI{writer: w, useColor: useColor}
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
	u.PrintlnColored("Error: "+message, ColorLightOrange)
}

func (u *UI) Success(message string) {
	u.PrintlnColored(message, ColorLightGreen)
}

func (u *UI) Warning(message string) {
	u.PrintlnColored("Warning: "+message, ColorLightYellow)
}

func (u *UI) Info(message string) {
	u.PrintlnColored(message, ColorLightBlue)
}

func (u *UI) GetPromptString(user, mindmap string) string {
	var promptBuilder strings.Builder
	promptBuilder.WriteString(u.colorize(user, ColorLightBlue))
	if mindmap != "" {
		promptBuilder.WriteString(u.colorize(" @ ", ColorWhite))
		promptBuilder.WriteString(u.colorize(mindmap, ColorPink))
	}
	promptBuilder.WriteString(u.colorize(" > ", ColorGreen))
	return promptBuilder.String()
}

func (u *UI) PrintCommand(command string) {
	u.PrintlnColored(command, ColorWhite)
}
