package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kagent.dev/kmcp/api/v1alpha1"
	"kagent.dev/kmcp/pkg/manifest"
	"sigs.k8s.io/yaml"
)

const (
	transportHTTP  = "http"
	transportStdio = "stdio"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [name]",
	Short: "Deploy MCP server to Kubernetes",
	Long: `Deploy an MCP server to Kubernetes by generating MCPServer CRDs.

This command generates MCPServer Custom Resource Definitions (CRDs) based on:
- Project configuration from kmcp.yaml
- Docker image built with 'kmcp build --docker'
- Deployment configuration options

The generated MCPServer will include:
- Docker image reference from the build
- Transport configuration (stdio/http)
- Port and command configuration
- Environment variables and secrets

Examples:
  kmcp deploy                          # Deploy with project name to cluster
  kmcp deploy my-server                # Deploy with custom name
  kmcp deploy --namespace staging      # Deploy to staging namespace
  kmcp deploy --dry-run                # Generate manifest without applying to cluster
  kmcp deploy --image custom:tag       # Use custom image
  kmcp deploy --transport http         # Use HTTP transport
  kmcp deploy --output deploy.yaml     # Save to file
  kmcp deploy --file /path/to/kmcp.yaml # Use custom kmcp.yaml file
  kmcp deploy --deploy-controller      # Deploy controller
  kmcp deploy --deploy-controller --controller-version 0.0.1 # Deploy controller with specific version
  kmcp deploy --deploy-controller --controller-namespace my-namespace # Deploy controller to custom namespace
  kmcp deploy --deploy-controller --registry-config ~/.docker/config.json # Specify docker registry config`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDeploy,
}

var (
	deployNamespace           string
	deployDryRun              bool
	deployOutput              string
	deployImage               string
	deployTransport           string
	deployPort                int
	deployTargetPort          int
	deployCommand             string
	deployArgs                []string
	deployEnv                 []string
	deployForce               bool
	deployFile                string
	deployController          bool
	deployControllerVersion   string
	deployControllerNamespace string
	deployRegistryConfig      string
)

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&deployNamespace, "namespace", "n", "default", "Kubernetes namespace")
	deployCmd.Flags().BoolVar(&deployDryRun, "dry-run", false, "Generate manifest without applying to cluster")
	deployCmd.Flags().StringVarP(&deployOutput, "output", "o", "", "Output file for the generated YAML")
	deployCmd.Flags().StringVar(&deployImage, "image", "", "Docker image to deploy (overrides build image)")
	deployCmd.Flags().StringVar(&deployTransport, "transport", "", "Transport type (stdio, http)")
	deployCmd.Flags().IntVar(&deployPort, "port", 0, "Container port (default: from project config)")
	deployCmd.Flags().IntVar(&deployTargetPort, "target-port", 0, "Target port for HTTP transport")
	deployCmd.Flags().StringVar(&deployCommand, "command", "", "Command to run (overrides project config)")
	deployCmd.Flags().StringSliceVar(&deployArgs, "args", []string{}, "Command arguments")
	deployCmd.Flags().StringSliceVar(&deployEnv, "env", []string{}, "Environment variables (KEY=VALUE)")
	deployCmd.Flags().BoolVar(&deployForce, "force", false, "Force deployment even if validation fails")
	deployCmd.Flags().StringVarP(&deployFile, "file", "f", "", "Path to kmcp.yaml file (default: current directory)")
	deployCmd.Flags().BoolVar(&deployController, "deploy-controller", false, "Deploy the KMCP controller to the cluster")
	deployCmd.Flags().StringVar(
		&deployControllerVersion,
		"controller-version",
		"",
		"Version of the controller to deploy (defaults to kmcp version)",
	)
	deployCmd.Flags().StringVar(
		&deployControllerNamespace,
		"controller-namespace",
		"kmcp-system",
		"Namespace for the KMCP controller (defaults to kmcp-system)",
	)
	// TODO: this var is currently required because the controller img is in a private registry but this may change
	deployCmd.Flags().StringVar(&deployRegistryConfig, "registry-config", "", "Path to docker registry config file")
}

func runDeploy(_ *cobra.Command, args []string) error {
	// Determine project directory
	var projectDir string
	var err error

	if deployFile != "" {
		// Use specified file path
		projectDir, err = getProjectDirFromFile(deployFile)
		if err != nil {
			return fmt.Errorf("failed to get project directory from file: %w", err)
		}
	} else {
		// Use current working directory
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Load project manifest
	manifestManager := manifest.NewManager(projectDir)
	if !manifestManager.Exists() {
		return fmt.Errorf("kmcp.yaml not found in %s. Run 'kmcp init' first or specify a valid path with --file", projectDir)
	}

	projectManifest, err := manifestManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load project manifest: %w", err)
	}

	// Deploy controller if requested
	if deployController {
		if err := deployControllerToCluster(); err != nil {
			return fmt.Errorf("failed to deploy controller: %w", err)
		}
	}

	// Determine deployment name
	deploymentName := projectManifest.Name
	if len(args) > 0 {
		deploymentName = args[0]
	}

	// Generate MCPServer resource
	mcpServer, err := generateMCPServer(projectManifest, deploymentName)
	if err != nil {
		return fmt.Errorf("failed to generate MCPServer: %w", err)
	}

	// Set namespace
	mcpServer.Namespace = deployNamespace

	if verbose {
		fmt.Printf("Generated MCPServer: %s/%s\n", mcpServer.Namespace, mcpServer.Name)
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(mcpServer)
	if err != nil {
		return fmt.Errorf("failed to marshal MCPServer to YAML: %w", err)
	}

	// Add YAML document separator and standard header
	yamlContent := fmt.Sprintf("---\n# MCPServer deployment generated by kmcp deploy\n# Project: %s\n# Framework: %s\n%s",
		projectManifest.Name, projectManifest.Framework, string(yamlData))

	// Handle output
	if deployOutput != "" {
		// Write to file
		if err := os.WriteFile(deployOutput, []byte(yamlContent), 0644); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
		fmt.Printf("✅ MCPServer manifest written to: %s\n", deployOutput)
	}

	if deployDryRun {
		// Print to stdout
		fmt.Print(yamlContent)
	} else {
		if err := applyToCluster(yamlContent, deploymentName); err != nil {
			return fmt.Errorf("failed to apply to cluster: %w", err)
		}
	}

	return nil
}

// getProjectDirFromFile extracts the project directory from a file path
func getProjectDirFromFile(filePath string) (string, error) {
	// Get absolute path of the file
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get the directory containing the file
	projectDir := filepath.Dir(absPath)

	// Verify the file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", absPath)
	}

	return projectDir, nil
}

func generateMCPServer(projectManifest *manifest.ProjectManifest, deploymentName string) (*v1alpha1.MCPServer, error) {
	// Determine image name
	imageName := deployImage
	if imageName == "" {
		// Use image from build config or generate default
		if projectManifest.Build.Docker.Image != "" {
			imageName = projectManifest.Build.Docker.Image
		} else {
			// Generate default image name
			imageName = fmt.Sprintf("%s:latest", strings.ToLower(strings.ReplaceAll(projectManifest.Name, "_", "-")))
		}
	}

	// Determine transport type
	transportType := v1alpha1.TransportTypeStdio
	if deployTransport != "" {
		switch deployTransport {
		case transportHTTP:
			transportType = v1alpha1.TransportTypeHTTP
		case transportStdio:
			transportType = v1alpha1.TransportTypeStdio
		default:
			return nil, fmt.Errorf("invalid transport type: %s (must be 'stdio' or 'http')", deployTransport)
		}
	}

	// Determine port
	port := deployPort
	if port == 0 {
		if projectManifest.Build.Docker.Port != 0 {
			port = projectManifest.Build.Docker.Port
		} else {
			port = 3000 // Default port
		}
	}

	// Determine command and args
	command := deployCommand
	args := deployArgs
	if command == "" {
		// Set default command based on framework
		command = getDefaultCommand(projectManifest.Framework)
		if len(args) == 0 {
			args = getDefaultArgs(projectManifest.Framework)
		}
	}

	// Parse environment variables
	envVars := parseEnvVars(deployEnv)

	// Add framework-specific environment variables
	for k, v := range projectManifest.Build.Docker.Environment {
		if envVars[k] == "" { // Don't override user-provided values
			envVars[k] = v
		}
	}

	// Create MCPServer spec
	mcpServer := &v1alpha1.MCPServer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kagent.dev/v1alpha1",
			Kind:       "MCPServer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentName,
			Labels: map[string]string{
				"app.kubernetes.io/name":       deploymentName,
				"app.kubernetes.io/instance":   deploymentName,
				"app.kubernetes.io/component":  "mcp-server",
				"app.kubernetes.io/part-of":    "kmcp",
				"app.kubernetes.io/managed-by": "kmcp",
				"kmcp.dev/framework":           projectManifest.Framework,
				"kmcp.dev/version":             projectManifest.Version,
			},
			Annotations: map[string]string{
				"kmcp.dev/project-name": projectManifest.Name,
				"kmcp.dev/description":  projectManifest.Description,
			},
		},
		Spec: v1alpha1.MCPServerSpec{
			Deployment: v1alpha1.MCPServerDeployment{
				Image: imageName,
				Port:  uint16(port),
				Cmd:   command,
				Args:  args,
				Env:   envVars,
			},
			TransportType: transportType,
		},
	}

	// Configure transport-specific settings
	if transportType == v1alpha1.TransportTypeHTTP {
		targetPort := deployTargetPort
		if targetPort == 0 {
			targetPort = port
		}
		mcpServer.Spec.HTTPTransport = &v1alpha1.HTTPTransport{
			TargetPort: uint32(targetPort),
			TargetPath: "/mcp",
		}
	} else {
		mcpServer.Spec.StdioTransport = &v1alpha1.StdioTransport{}
	}

	return mcpServer, nil
}

func getDefaultCommand(framework string) string {
	switch framework {
	case manifest.FrameworkFastMCPPython:
		return "python"
	default:
		return "python"
	}
}

func getDefaultArgs(framework string) []string {
	switch framework {
	case manifest.FrameworkFastMCPPython:
		return []string{"src/main.py"}
	default:
		return []string{"src/main.py"}
	}
}

func parseEnvVars(envVars []string) map[string]string {
	result := make(map[string]string)
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func applyToCluster(yamlContent, deploymentName string) error {
	fmt.Printf("🚀 Applying MCPServer to cluster...\n")

	// Check if kubectl is available
	if err := checkKubectlAvailable(); err != nil {
		return fmt.Errorf("kubectl is required for cluster deployment: %w", err)
	}

	// Create temporary file for kubectl apply
	tmpFile, err := os.CreateTemp("", "mcpserver-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write YAML content to temp file
	if _, err := tmpFile.Write([]byte(yamlContent)); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Apply using kubectl
	if err := runKubectl("apply", "-f", tmpFile.Name()); err != nil {
		return fmt.Errorf("kubectl apply failed: %w", err)
	}

	fmt.Printf("✅ MCPServer '%s' applied successfully\n", deploymentName)
	fmt.Printf("💡 Check status with: kubectl get mcpserver %s -n %s\n", deploymentName, deployNamespace)
	fmt.Printf("💡 View logs with: kubectl logs -l app.kubernetes.io/name=%s -n %s\n", deploymentName, deployNamespace)

	if err := os.Remove(tmpFile.Name()); err != nil {
		fmt.Printf("failed to remove temp file: %v\n", err)
	}
	return nil
}

func runKubectl(args ...string) error {
	if verbose {
		fmt.Printf("Running: kubectl %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// getKMCPVersion returns the current kmcp version
func getKMCPVersion() string {
	return Version
}

// deployControllerToCluster deploys the kmcp controller using helm
func deployControllerToCluster() error {
	fmt.Printf("🚀 Deploying KMCP controller to cluster...\n")

	// Check if helm is available
	if err := checkHelmAvailable(); err != nil {
		return fmt.Errorf("helm is required for controller deployment: %w", err)
	}

	// Determine controller version
	controllerVersion := deployControllerVersion
	if controllerVersion == "" {
		controllerVersion = getKMCPVersion()
	}

	// Validate controller version format
	if controllerVersion == "" {
		return fmt.Errorf("invalid controller version: version cannot be empty")
	}

	// Determine registry config file
	registryConfig := deployRegistryConfig
	if registryConfig == "" {
		fmt.Print("Docker registry config must be set use --registry-config\n")
	}
	if registryConfig != "" && verbose {
		fmt.Printf("Using registry config: %s\n", registryConfig)
	}

	// Build helm install command
	args := []string{
		"install", "kmcp", "oci://ghcr.io/kagent-dev/kmcp/helm/kmcp",
		"--version", controllerVersion,
		"--namespace", deployControllerNamespace,
		"--create-namespace",
	}

	// Add registry config if found
	if registryConfig != "" {
		args = append(args, "--registry-config", registryConfig)
	}

	// Run helm install
	if err := runHelm(args...); err != nil {
		return fmt.Errorf("helm install failed: %w", err)
	}

	fmt.Printf(
		"✅ KMCP controller deployed successfully with version %s\n",
		controllerVersion,
	)
	fmt.Printf(
		"💡 Check controller status with: kubectl get pods -n %s\n",
		deployControllerNamespace,
	)
	fmt.Printf(
		"💡 View controller logs with: kubectl logs -l app.kubernetes.io/name=kmcp-controller-manager -n %s\n",
		deployControllerNamespace,
	)

	return nil
}

// checkKubectlAvailable checks if kubectl is available in the system
func checkKubectlAvailable() error {
	cmd := exec.Command("kubectl", "version", "--client")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl not found or not working: %w", err)
	}
	return nil
}

// checkHelmAvailable checks if helm is available in the system
func checkHelmAvailable() error {
	cmd := exec.Command("helm", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm not found or not working: %w", err)
	}
	return nil
}

// runHelm executes helm commands
func runHelm(args ...string) error {
	if verbose {
		fmt.Printf("Running: helm %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
