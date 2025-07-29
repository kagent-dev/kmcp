package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

var kindCmd = &cobra.Command{
	Use:   "kind",
	Short: "Run MCP server in kind cluster",
	Long: `Run an MCP server in a kind cluster by building and deploying.

This command will:
1. Check if kind is available and create a cluster if needed
2. Deploy the KMCP controller (includes CRDs)
3. Build the MCP server Docker image
4. Deploy the MCP server to the kind cluster in the specified namespace

Examples:
  kmcp run kind --project-dir ./my-project  # Run from specific directory
  kmcp run kind --namespace my-namespace    # Deploy to specific namespace
  kmcp run kind --registry-config ~/.docker/config.json  # Use custom registry config
  kmcp run kind --version v0.0.1      # Deploy specific controller version`,
	RunE: executeKind,
}

var (
	localProjectDir    string
	kindNamespace      string
	kindRegistryConfig string
	kindVersion        string
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.AddCommand(localCmd)
	runCmd.AddCommand(kindCmd)

	localCmd.Flags().StringVarP(&localProjectDir, "project-dir", "d", "", "Project directory to use (default: current directory)")
	kindCmd.Flags().StringVarP(&localProjectDir, "project-dir", "d", "", "Project directory to use (default: current directory)")
	kindCmd.Flags().StringVarP(&kindNamespace, "namespace", "n", "default", "Namespace to deploy to (default: default)")
	// TODO: registry-config flag can be removed once the helm chart is in a publiclly accessible registry
	kindCmd.Flags().StringVar(&kindRegistryConfig, "registry-config", "", "Path to docker registry config file (required for controller deployment)")
	kindCmd.Flags().StringVar(&kindVersion, "version", "", "Version of the controller to deploy (defaults to kmcp version)")
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

func executeKind(_ *cobra.Command, _ []string) error {
	projectDir, err := getProjectDir()
	if err != nil {
		return err
	}

	manifest, err := getProjectManifest(projectDir)
	if err != nil {
		return err
	}

	// Check if kind is available
	if err := checkKindAvailable(); err != nil {
		return err
	}

	// Ensure kind cluster exists
	if err := ensureKindCluster(); err != nil {
		return err
	}

	// Deploy controller
	if err := deployController(kindNamespace, kindRegistryConfig, kindVersion); err != nil {
		return err
	}

	// Build the Docker image
	if err := buildDockerImage(projectDir); err != nil {
		return err
	}

	// Deploy to kind cluster
	if err := deployToKind(projectDir, manifest, kindNamespace); err != nil {
		return err
	}

	fmt.Printf("✅ MCP server successfully deployed to kind cluster\n")
	fmt.Printf("💡 Check status with: kubectl get mcpserver %s -n %s\n", manifest.Name, kindNamespace)
	fmt.Printf("💡 View logs with: kubectl logs -l app.kubernetes.io/name=%s -n %s\n", manifest.Name, kindNamespace)
	fmt.Printf("🔌 Port forward the MCP server: kubectl port-forward deployment/%s 3000:3000 -n %s\n", manifest.Name, kindNamespace)
	fmt.Printf("\nPress Enter to start the MCP inspector...")
	fmt.Scanln() // Wait for user input

	// Create MCP inspector config and start inspector
	if err := startInspector(projectDir, manifest.Name); err != nil {
		return err
	}

	return nil
}

func checkKindAvailable() error {
	cmd := exec.Command("kind", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kind is required but not found. Please install kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation")
	}
	return nil
}

func ensureKindCluster() error {
	// Check if kind cluster exists
	cmd := exec.Command("kind", "get", "clusters")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check kind clusters: %w", err)
	}

	clusters := strings.TrimSpace(string(output))
	if !strings.Contains(clusters, "kind") {
		if verbose {
			fmt.Printf("Creating kind cluster...\n")
		}

		// Create kind cluster
		createCmd := exec.Command("kind", "create", "cluster", "--name", "kind")
		createCmd.Stdout = os.Stdout
		createCmd.Stderr = os.Stderr
		if err := createCmd.Run(); err != nil {
			return fmt.Errorf("failed to create kind cluster: %w", err)
		}
	} else if verbose {
		fmt.Printf("Kind cluster already exists\n")
	}

	return nil
}

func buildDockerImage(projectDir string) error {
	if verbose {
		fmt.Printf("Building Docker image...\n")
	}

	// Run kmcp build --docker
	buildCmd := exec.Command("kmcp", "build", "--docker", "--project-dir", projectDir)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	// Load the built image into kind cluster
	if verbose {
		fmt.Printf("Loading MCP server image into kind cluster\n")
	}

	// Get the project name from the manifest to determine the image name
	manifest, err := getProjectManifest(projectDir)
	if err != nil {
		return fmt.Errorf("failed to get project manifest: %w", err)
	}

	// Use the project name as the image name (this is the default behavior of kmcp build)
	imageName := fmt.Sprintf("%s:latest", strings.ToLower(strings.ReplaceAll(manifest.Name, "_", "-")))

	loadCmd := exec.Command("kind", "load", "docker-image", imageName)
	loadCmd.Stdout = os.Stdout
	loadCmd.Stderr = os.Stderr
	if err := loadCmd.Run(); err != nil {
		return fmt.Errorf("failed to load MCP server image into kind cluster: %w", err)
	}

	return nil
}

func deployToKind(projectDir string, manifest *manifest.ProjectManifest, namespace string) error {
	if verbose {
		fmt.Printf("Deploying to kind cluster in namespace: %s\n", namespace)
	}

	// Run kmcp deploy mcp with namespace
	deployCmd := exec.Command("kmcp", "deploy", "mcp", manifest.Name, "--file", fmt.Sprintf("%s/kmcp.yaml", projectDir), "--namespace", namespace)
	deployCmd.Stdout = os.Stdout
	deployCmd.Stderr = os.Stderr
	if err := deployCmd.Run(); err != nil {
		return fmt.Errorf("failed to deploy to kind cluster: %w", err)
	}

	return nil
}

func installCRDs() error {
	if verbose {
		fmt.Printf("Installing CRDs...\n")
	}

	// Apply CRDs from the config directory
	cmd := exec.Command("kubectl", "apply", "-f", "config/crd/bases/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install CRDs: %w", err)
	}

	return nil
}

func deployController(namespace string, registryConfig string, version string) error {
	if verbose {
		fmt.Printf("Deploying controller to namespace: %s\n", namespace)
		if version != "" {
			fmt.Printf("Using specified version: %s\n", version)
		}
	}

	// Pull and load the controller image into kind cluster
	if version != "" {
		imageName := fmt.Sprintf("ghcr.io/kagent-dev/kmcp/controller:%s", version)
		if verbose {
			fmt.Printf("Pulling controller image: %s\n", imageName)
		}

		// Pull the image
		pullCmd := exec.Command("docker", "pull", imageName)
		pullCmd.Stdout = os.Stdout
		pullCmd.Stderr = os.Stderr
		if err := pullCmd.Run(); err != nil {
			return fmt.Errorf("failed to pull controller image: %w", err)
		}

		// Load the image into kind cluster
		if verbose {
			fmt.Printf("Loading controller image into kind cluster\n")
		}
		loadCmd := exec.Command("kind", "load", "docker-image", imageName)
		loadCmd.Stdout = os.Stdout
		loadCmd.Stderr = os.Stderr
		if err := loadCmd.Run(); err != nil {
			return fmt.Errorf("failed to load controller image into kind cluster: %w", err)
		}
	}

	// Build helm install command directly
	args := []string{
		"install", "kmcp", "oci://ghcr.io/kagent-dev/kmcp/helm/kmcp",
		"--version", version,
		"--namespace", namespace,
		"--create-namespace",
	}

	// Add registry config if found
	if registryConfig != "" {
		args = append(args, "--registry-config", registryConfig)
	}

	// Override the image tag to match the version
	if version != "" {
		args = append(args, "--set", fmt.Sprintf("image.tag=%s", version))
	}

	if verbose {
		fmt.Printf("Executing: helm %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to deploy controller: %w", err)
	}

	return nil
}

func startInspector(projectDir string, serverName string) error {
	if verbose {
		fmt.Printf("Starting MCP inspector...\n")
	}

	// Create server configuration for kind deployment
	serverConfig := map[string]interface{}{
		"type": "streamable-http",
		"url":  "http://localhost:3000/mcp",
	}

	// Create MCP inspector config
	configPath := filepath.Join(projectDir, "mcp-server-config.json")
	if err := createMCPInspectorConfig(serverName, serverConfig, configPath); err != nil {
		return err
	}

	// Run the inspector in background
	return runMCPInspector(configPath, serverName, projectDir)
}
