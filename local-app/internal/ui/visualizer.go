// Package ui provides user interface functionality for the Mindnoscape application.
// This file contains the Visualizer struct and methods for handling colored output.
package ui

import (
	"fmt"
	"io"
	"strings"
)

// Visualizer handles the low-level output formatting and coloring.
type Visualizer struct {
	writer   io.Writer
	useColor bool
}

// NewVisualizer creates a new Visualizer instance.
func NewVisualizer(w io.Writer, useColor bool) *Visualizer {
	return &Visualizer{
		writer:   w,
		useColor: useColor,
	}
}

// Print writes a message to the Visualizer's writer.
func (v *Visualizer) Print(message string) {
	fmt.Fprint(v.writer, message)
}

// Printf writes a formatted message to the Visualizer's writer.
func (v *Visualizer) Printf(format string, args ...interface{}) {
	fmt.Fprintf(v.writer, format, args...)
}

// Println writes a message followed by a newline to the Visualizer's writer.
func (v *Visualizer) Println(message string) {
	fmt.Fprintln(v.writer, message)
}

// PrintColored writes a colored message to the Visualizer's writer if color is enabled.
func (v *Visualizer) PrintColored(message string, color Color) {
	if v.useColor {
		fmt.Fprintf(v.writer, "%s%s%s", color, message, ColorDefault)
	} else {
		fmt.Fprint(v.writer, message)
	}
}

// PrintMultiColoredLine prints a line with multiple color codes.
func (v *Visualizer) PrintMultiColoredLine(line string, colorMap map[string]Color) {
	// Process the line in parts, handling color codes
	for len(line) > 0 {
		// Find the next color code
		startIndex := strings.Index(line, "{{")
		if startIndex == -1 {
			v.Print(line)
			break
		}

		endIndex := strings.Index(line, "}}")
		if endIndex == -1 {
			v.Print(line)
			break
		}

		// Print the part before the color code
		if startIndex > 0 {
			v.Print(line[:startIndex])
		}

		// Extract and apply the color code
		colorCode := line[startIndex : endIndex+2]
		color, exists := colorMap[colorCode]
		if !exists {
			color = ColorDefault
		}

		// Find the next color code or the end of the string
		nextStartIndex := strings.Index(line[endIndex+2:], "{{")
		if nextStartIndex == -1 {
			// No more color codes, print the rest of the line
			v.PrintColored(line[endIndex+2:], color)
			break
		} else {
			// Print the part until the next color code
			v.PrintColored(line[endIndex+2:endIndex+2+nextStartIndex], color)
			line = line[endIndex+2+nextStartIndex:]
		}
	}
	v.Println("") // New line at the end
}
