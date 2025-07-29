package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"kagent.dev/kmcp/pkg/manifest"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run MCP server",
	Long: `Run an MCP server using the Model Context Protocol inspector.

This command provides subcommands for different deployment scenarios.`,
}

var localCmd = &cobra.Command{
	Use:   "local",
	Short: "Run MCP server locally",
	Long: `Run an MCP server locally using the Model Context Protocol inspector.

This command will:
1. Load the kmcp.yaml configuration from the project directory
2. Determine the framework type and create appropriate configuration
3. Run the MCP server using the Model Context Protocol inspector

Examples:
  kmcp run local --project-dir ./my-project  # Run from specific directory`,
	RunE: executeLocal,
}

var (
	localProjectDir string
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.AddCommand(localCmd)

	localCmd.Flags().StringVarP(
		&localProjectDir,
		"project-dir",
		"d",
		"",
		"Project directory to use (default: current directory)",
	)
}

func executeLocal(_ *cobra.Command, _ []string) error {
	projectDir, err := getProjectDir()
	if err != nil {
		return err
	}

	manifest, err := getProjectManifest(projectDir)
	if err != nil {
		return err
	}

	// Check if npx is installed
	if err := checkNpxInstalled(); err != nil {
		return err
	}

	// Determine framework and create configuration
	switch manifest.Framework {
	case "fastmcp-python":
		return runFastMCPPython(projectDir, manifest)
	default:
		return fmt.Errorf("unsupported framework: %s", manifest.Framework)
	}
}

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

func runFastMCPPython(projectDir string, manifest *manifest.ProjectManifest) error {
	// Check if uv is available
	if _, err := exec.LookPath("uv"); err != nil {
		uvInstallURL := "https://docs.astral.sh/uv/getting-started/installation/"
		return fmt.Errorf(
			"uv is required for this command to run fastmcp-python projects locally. Please install uv: %s", uvInstallURL,
		)
	}

	// Run uv sync first
	if verbose {
		fmt.Printf("Running uv sync in: %s\n", projectDir)
	}
	syncCmd := exec.Command("uv", "sync")
	syncCmd.Dir = projectDir
	syncCmd.Stdout = os.Stdout
	syncCmd.Stderr = os.Stderr
	if err := syncCmd.Run(); err != nil {
		return fmt.Errorf("failed to run uv sync: %w", err)
	}

	// Create server configuration for local execution
	serverConfig := map[string]interface{}{
		"command": "uv",
		"args":    []string{"run", "python", "src/main.py"},
	}

	// Create MCP inspector config
	configPath := filepath.Join(projectDir, "mcp-server-config.json")
	if err := createMCPInspectorConfig(manifest.Name, serverConfig, configPath); err != nil {
		return err
	}

	// Run the inspector
	return runMCPInspector(configPath, manifest.Name, projectDir)
}

func getProjectDir() (string, error) {
	// Determine project directory
	projectDir := localProjectDir
	if projectDir == "" {
		// Use current working directory
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	} else {
		// Convert relative path to absolute path
		if !filepath.IsAbs(projectDir) {
			cwd, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("failed to get current directory: %w", err)
			}
			projectDir = filepath.Join(cwd, projectDir)
		}
	}

	if verbose {
		fmt.Printf("Using project directory: %s\n", projectDir)
	}

	return projectDir, nil
}

func getProjectManifest(projectDir string) (*manifest.ProjectManifest, error) {
	// Check if kmcp.yaml exists
	manager := manifest.NewManager(projectDir)
	if !manager.Exists() {
		return nil, fmt.Errorf("this directory is not an mcp-server directory: kmcp.yaml not found")
	}

	// Load the manifest
	manifest, err := manager.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load kmcp.yaml: %w", err)
	}

	return manifest, nil
}
