package ui

import (
	"fmt"
	"io"
	"strings"
)

type Visualizer struct {
	writer   io.Writer
	useColor bool
}

func NewVisualizer(w io.Writer, useColor bool) *Visualizer {
	return &Visualizer{
		writer:   w,
		useColor: useColor,
	}
}

func (v *Visualizer) Print(message string) {
	fmt.Fprint(v.writer, message)
}

func (v *Visualizer) Printf(format string, args ...interface{}) {
	fmt.Fprintf(v.writer, format, args...)
}

func (v *Visualizer) Println(message string) {
	fmt.Fprintln(v.writer, message)
}

func (v *Visualizer) PrintColored(message string, color Color) {
	if v.useColor {
		fmt.Fprintf(v.writer, "%s%s%s", color, message, ColorDefault)
	} else {
		fmt.Fprint(v.writer, message)
	}
}

func (v *Visualizer) PrintMultiColoredLine(line string, colorMap map[string]Color) {
	for len(line) > 0 {
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
