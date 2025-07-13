package main

import (
	"fmt"
	"os"

	"kagent.dev/kmcp/cmd/kmcp/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
