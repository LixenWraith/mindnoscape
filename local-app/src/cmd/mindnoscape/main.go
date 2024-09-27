// Package main is the entry point for the Mindnoscape application.
// It delegates all the component initialization to bootstrap function.
package main

import (
	"fmt"
	"os"
)

// main is the entry point of the application.
func main() {
	if err := bootstrap(); err != nil {
		fmt.Printf("Error bootstrapping the application: %v\n", err)
		os.Exit(1)
	}
}
