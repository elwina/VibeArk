package main

import (
	"fmt"
	"os"

	vibearkcmd "vibeark/cmd"
)

func main() {
	if err := vibearkcmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
