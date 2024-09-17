package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorWhite  = "\033[37m"
)

func printHelp() {
	fmt.Println("Usage: logviewer [log directory] [-r <refresh rate in seconds>] [-h|--help]")
	fmt.Println("\nOptions:")
	fmt.Println("  [log directory]      Path to the directory containing log files (default: ./logs/)")
	fmt.Println("  -r, --rate           Refresh rate in seconds (default: 1)")
	fmt.Println("  -h, --help           Show this help message")
	fmt.Println("\nDescription:")
	fmt.Println("  This tool monitors all *.log files in the specified directory.")
	fmt.Println("  Timestamps are displayed in yellow, info messages in white,")
	fmt.Println("  command logs in blue, and error logs in red.")
	fmt.Println("\nExample:")
	fmt.Println("  logviewer /path/to/logs -r 2")
}

type LogEntry struct {
	Time  string `json:"time"`
	Level string `json:"level"`
	Msg   string `json:"msg"`
}

func formatTimestamp(timestamp string) string {
	t, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return timestamp // Return original if parsing fails
	}
	return t.Format("06-01-02 15:04:05.000000")
}

func main() {
	var refreshRate int
	var help bool

	flag.IntVar(&refreshRate, "r", 1, "Refresh rate in seconds")
	flag.IntVar(&refreshRate, "rate", 1, "Refresh rate in seconds")
	flag.BoolVar(&help, "h", false, "Show help")
	flag.BoolVar(&help, "help", false, "Show help")
	flag.Parse()

	if help {
		printHelp()
		os.Exit(0)
	}

	args := flag.Args()
	logDir := "./logs/"
	if len(args) > 0 {
		fileInfo, err := os.Stat(args[0])
		if err == nil && fileInfo.IsDir() {
			logDir = args[0]
		} else {
			fmt.Printf("WARNING: '%s' is not a valid directory. Using default './logs/'\n", args[0])
		}
	}

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		fmt.Printf("Log directory '%s' does not exist. Please specify a valid directory.\n", logDir)
		os.Exit(1)
	}

	fmt.Printf("Monitoring logs in directory: %s\n", logDir)
	fmt.Printf("Refresh rate: %d second(s)\n", refreshRate)

	filePositions := make(map[string]int64)
	knownFiles := make(map[string]bool)

	for {
		logFiles, err := filepath.Glob(filepath.Join(logDir, "*.log"))
		if err != nil {
			fmt.Printf("%sError reading log directory: %v%s\n", colorRed, err, colorReset)
			time.Sleep(time.Duration(refreshRate) * time.Second)
			continue
		}

		// Detect new files
		for _, filePath := range logFiles {
			if !knownFiles[filePath] {
				fmt.Printf("%sNew log file detected: %s%s\n", colorGreen, filepath.Base(filePath), colorReset)
				knownFiles[filePath] = true
			}
		}

		for _, filePath := range logFiles {
			fileName := filepath.Base(filePath)

			file, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
			if err != nil {
				fmt.Printf("%sError opening %s: %v%s\n", colorRed, fileName, err, colorReset)
				continue
			}

			stat, err := file.Stat()
			if err != nil {
				fmt.Printf("%sError getting file stats for %s: %v%s\n", colorRed, fileName, err, colorReset)
				file.Close()
				continue
			}

			if stat.Size() < filePositions[filePath] {
				fmt.Printf("%s%s has been truncated, starting from beginning%s\n", colorYellow, fileName, colorReset)
				filePositions[filePath] = 0
			}

			_, err = file.Seek(filePositions[filePath], io.SeekStart)
			if err != nil {
				fmt.Printf("%sError seeking in %s: %v%s\n", colorRed, fileName, err, colorReset)
				file.Close()
				continue
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				var entry LogEntry
				err := json.Unmarshal([]byte(line), &entry)
				if err != nil {
					fmt.Printf("%sError parsing log entry: %v%s\n", colorRed, err, colorReset)
					continue
				}

				// Print formatted timestamp in yellow
				fmt.Printf("%s%s%s", colorYellow, formatTimestamp(entry.Time), colorReset)

				// Print level if it exists and is not empty
				if entry.Level != "" && entry.Level != "INFO" {
					fmt.Printf(", %s", entry.Level)
				}

				// Determine message color based on log file type
				var messageColor string
				switch {
				case strings.Contains(fileName, "command"):
					messageColor = colorBlue
				case strings.Contains(fileName, "error"):
					messageColor = colorRed
				default:
					messageColor = colorWhite
				}

				// Print message
				if entry.Level != "" && entry.Level != "INFO" {
					fmt.Printf(", ")
				}
				fmt.Printf("%s%s%s\n", messageColor, entry.Msg, colorReset)
			}

			if err := scanner.Err(); err != nil {
				fmt.Printf("%sError reading %s: %v%s\n", colorRed, fileName, err, colorReset)
			}

			newPosition, err := file.Seek(0, io.SeekCurrent)
			if err != nil {
				fmt.Printf("%sError getting current position in %s: %v%s\n", colorRed, fileName, err, colorReset)
			} else {
				filePositions[filePath] = newPosition
			}

			file.Close()
		}

		time.Sleep(time.Duration(refreshRate) * time.Second)
	}
}
