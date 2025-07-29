package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// findProjectRoot finds the root of the KMCP project by looking for kmcp.yaml.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "kmcp.yaml")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "kmcp.yml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in a KMCP project directory")
		}
		dir = parent
	}
}

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func promptForInput(promptText string) (string, error) {
	fmt.Print(promptText)
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		if err.Error() == "unexpected newline" {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(input), nil
}
