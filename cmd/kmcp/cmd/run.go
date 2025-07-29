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

	localCmd.Flags().StringVarP(&localProjectDir, "project-dir", "d", "", "Project directory to use (required)")
	localCmd.MarkFlagRequired("project-dir")
}

func executeLocal(_ *cobra.Command, _ []string) error {
	// Determine project directory
	projectDir := localProjectDir
	if projectDir == "" {
		return fmt.Errorf("--project-dir is required")
	}

	// Convert relative path to absolute path
	if !filepath.IsAbs(projectDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = filepath.Join(cwd, projectDir)
	}

	if verbose {
		fmt.Printf("Using project directory: %s\n", projectDir)
	}

	// Check if kmcp.yaml exists
	manager := manifest.NewManager(projectDir)
	if !manager.Exists() {
		return fmt.Errorf("this directory is not an mcp-server directory: kmcp.yaml not found")
	}

	// Load the manifest
	manifest, err := manager.Load()
	if err != nil {
		return fmt.Errorf("failed to load kmcp.yaml: %w", err)
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

func runFastMCPPython(projectDir string, manifest *manifest.ProjectManifest) error {
	// Check if uv is available
	if _, err := exec.LookPath("uv"); err != nil {
		return fmt.Errorf("uv is required for this commandto run fastmcp-python projects locally. Please install uv: https://docs.astral.sh/uv/getting-started/installation/")
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

	// Create mcp-server-config.json
	var serverConfig map[string]interface{}
	serverConfig = map[string]interface{}{
		"command": "uv",
		"args":    []string{"run", "python", "src/main.py"},
	}

	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			manifest.Name: serverConfig,
		},
	}

	configPath := filepath.Join(projectDir, "mcp-server-config.json")
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

	// Run the inspector
	args := []string{
		"@modelcontextprotocol/inspector",
		"--config", configPath,
		"--server", manifest.Name,
	}

	if verbose {
		fmt.Printf("Running: npx %s\n", args)
	}

	cmd := exec.Command("npx", args...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
