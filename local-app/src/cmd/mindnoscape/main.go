package main

import (
	"fmt"
	"os"
)

func main() {
	if err := bootstrap(); err != nil {
		fmt.Printf("Error bootstrapping the application: %v\n", err)
		os.Exit(1)
	}
}
