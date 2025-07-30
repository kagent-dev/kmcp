package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

func checkNpxInstalled() error {
	cmd := exec.Command("npx", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npx is required to run the modelcontextinstaller. Please install Node.js and npm to get npx")
	}
	return nil
}

// createMCPInspectorConfig creates an MCP inspector configuration file
func createMCPInspectorConfig(serverName string, serverConfig map[string]interface{}, configPath string) error {
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			serverName: serverConfig,
		},
	}

	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write mcp-server-config.json: %w", err)
	}

	if verbose {
		fmt.Printf("Created mcp-server-config.json: %s\n", configPath)
	}

	return nil
}

// runMCPInspector runs the MCP inspector with the given configuration
func runMCPInspector(configPath, serverName string, workingDir string) error {
	args := []string{
		"@modelcontextprotocol/inspector",
		"--config", configPath,
		"--server", serverName,
	}

	if verbose {
		fmt.Printf("Running: npx %s\n", args)
	}

	cmd := exec.Command("npx", args...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run synchronously
	return cmd.Run()
}
