package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kagent-dev/kmcp/pkg/manifest"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run MCP server locally",
	Long: `Run an MCP server locally using the Model Context Protocol inspector.

By default, this command will:
1. Load the kmcp.yaml configuration from the project directory
2. Determine the framework type and create the appropriate mcp inspector configuration
3. Launch the MCP inspector, which will start the server when you click "Connect"

If you want to run the server directly without the inspector, use the --no-inspector flag.
This will execute the server directly using the appropriate framework command.

Supported frameworks:
- fastmcp-python: Requires uv to be installed
- mcp-go: Requires Go to be installed

Examples:
  kmcp run --project-dir ./my-project     # Run with inspector (default)
  kmcp run --no-inspector                 # Run server directly without inspector`,
	RunE: executeRun,
}

var (
	projectDir  string
	noInspector bool
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(
		&projectDir,
		"project-dir",
		"d",
		"",
		"Project directory to use (default: current directory)",
	)
	runCmd.Flags().BoolVar(
		&noInspector,
		"no-inspector",
		false,
		"Run the server directly without launching the MCP inspector",
	)
}

func executeRun(_ *cobra.Command, _ []string) error {
	projectDir, err := getProjectDir()
	if err != nil {
		return err
	}

	manifest, err := getProjectManifest(projectDir)
	if err != nil {
		return err
	}

	// Check if npx is installed (only needed when using inspector)
	if !noInspector {
		if err := checkNpxInstalled(); err != nil {
			return err
		}
	}

	// Determine framework and create configuration
	switch manifest.Framework {
	case "fastmcp-python":
		return runFastMCPPython(projectDir, manifest)
	case "mcp-go":
		return runMCPGo(projectDir, manifest)
	default:
		return fmt.Errorf("unsupported framework: %s", manifest.Framework)
	}
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

	if noInspector {
		// Run the server directly
		fmt.Printf("Running server directly: uv run python src/main.py\n")
		fmt.Printf("Server is running and waiting for MCP protocol input on stdin...\n")
		fmt.Printf("Press Ctrl+C to stop the server\n")

		serverCmd := exec.Command("uv", "run", "python", "src/main.py")
		serverCmd.Dir = projectDir
		serverCmd.Stdout = os.Stdout
		serverCmd.Stderr = os.Stderr
		serverCmd.Stdin = os.Stdin
		return serverCmd.Run()
	}

	// Create server configuration for inspector
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

func runMCPGo(projectDir string, manifest *manifest.ProjectManifest) error {
	// Check if go is available
	if _, err := exec.LookPath("go"); err != nil {
		goInstallURL := "https://golang.org/doc/install"
		return fmt.Errorf("go is required to run mcp-go projects locally. Please install Go: %s", goInstallURL)
	}

	// Run go mod tidy first to ensure dependencies are up to date
	if verbose {
		fmt.Printf("Running go mod tidy in: %s\n", projectDir)
	}
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = projectDir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("failed to run go mod tidy: %w", err)
	}

	if noInspector {
		// Run the server directly
		fmt.Printf("Running server directly: go run main.go\n")
		fmt.Printf("Server is running and waiting for MCP protocol input on stdin...\n")
		fmt.Printf("Press Ctrl+C to stop the server\n")

		serverCmd := exec.Command("go", "run", "main.go")
		serverCmd.Dir = projectDir
		serverCmd.Stdout = os.Stdout
		serverCmd.Stderr = os.Stderr
		serverCmd.Stdin = os.Stdin
		return serverCmd.Run()
	}

	// Create server configuration for inspector
	serverConfig := map[string]interface{}{
		"command": "go",
		"args":    []string{"run", "main.go"},
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
	dir := projectDir
	if dir == "" {
		// Use current working directory
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	} else {
		// Convert relative path to absolute path
		if !filepath.IsAbs(dir) {
			cwd, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("failed to get current directory: %w", err)
			}
			dir = filepath.Join(cwd, dir)
		}
	}

	if verbose {
		fmt.Printf("Using project directory: %s\n", dir)
	}

	return dir, nil
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
