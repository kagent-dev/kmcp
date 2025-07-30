package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func promptForInput(promptText string) (string, error) {
	fmt.Print(promptText)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}
