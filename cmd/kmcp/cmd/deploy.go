package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kagent.dev/kmcp/api/v1alpha1"
	"kagent.dev/kmcp/pkg/manifest"
	"kagent.dev/kmcp/pkg/wellknown"
	"sigs.k8s.io/yaml"
)

const (
	transportHTTP  = "http"
	transportStdio = "stdio"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy to Kubernetes",
	Long: `Deploy components to Kubernetes clusters.

This command provides functionality to deploy MCP servers and the KMCP controller
to Kubernetes clusters.`,
}

// deployMCPCmd deploys an MCP server to Kubernetes
var deployMCPCmd = &cobra.Command{
	Use:   "mcp [name]",
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

The command can also apply Kubernetes secret YAML files to the cluster before deploying the MCPServer.
The secrets will be referenced in the MCPServer CRD for mounting as volumes to the MCP server container.
Secret namespace will be overridden with the deployment namespace to avoid the need for reference grants
to enable cross-namespace references.

Examples:
  kmcp deploy mcp                          # Deploy with project name to cluster
  kmcp deploy mcp my-server                # Deploy with custom name
  kmcp deploy mcp --namespace staging      # Deploy to staging namespace
  kmcp deploy mcp --dry-run                # Generate manifest without applying to cluster
  kmcp deploy mcp --image custom:tag       # Use custom image
  kmcp deploy mcp --transport http         # Use HTTP transport
  kmcp deploy mcp --output deploy.yaml     # Save to file
  kmcp deploy mcp --file /path/to/kmcp.yaml # Use custom kmcp.yaml file
  kmcp deploy mcp --secrets secret1.yaml,secret2.yaml # Apply secret files to cluster`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDeployMCP,
}

// deployControllerCmd deploys the KMCP controller to the cluster
var deployControllerCmd = &cobra.Command{
	Use:   "controller",
	Short: "Deploy KMCP controller to cluster",
	Long: `Deploy the KMCP controller to a Kubernetes cluster using Helm.

This command installs the KMCP controller from the Helm chart repository.
The controller manages MCPServer custom resources and handles their lifecycle.

Examples:
  kmcp deploy controller                                    # Deploy with default settings
  kmcp deploy controller --version 0.0.1                   # Deploy specific version
  kmcp deploy controller --namespace my-namespace          # Deploy to custom namespace
  kmcp deploy controller --registry-config ~/.docker/config.json # Use custom registry config`,
	RunE: runDeployController,
}

var (
	// MCP deployment flags
	deployNamespace  string
	deployDryRun     bool
	deployOutput     string
	deployImage      string
	deployTransport  string
	deployPort       int
	deployTargetPort int
	deployCommand    string
	deployArgs       []string
	deployEnv        []string
	deployForce      bool
	deployFile       string
	deploySecrets    []string

	// Controller deployment flags
	controllerVersion        string
	controllerNamespace      string
	controllerRegistryConfig string
)

func init() {
	rootCmd.AddCommand(deployCmd)

	// Add subcommands
	deployCmd.AddCommand(deployMCPCmd)
	deployCmd.AddCommand(deployControllerCmd)

	// MCP deployment flags
	deployMCPCmd.Flags().StringVarP(&deployNamespace, "namespace", "n", "default", "Kubernetes namespace")
	deployMCPCmd.Flags().BoolVar(&deployDryRun, "dry-run", false, "Generate manifest without applying to cluster")
	deployMCPCmd.Flags().StringVarP(&deployOutput, "output", "o", "", "Output file for the generated YAML")
	deployMCPCmd.Flags().StringVar(&deployImage, "image", "", "Docker image to deploy (overrides build image)")
	deployMCPCmd.Flags().StringVar(&deployTransport, "transport", "", "Transport type (stdio, http)")
	deployMCPCmd.Flags().IntVar(&deployPort, "port", 0, "Container port (default: from project config)")
	deployMCPCmd.Flags().IntVar(&deployTargetPort, "target-port", 0, "Target port for HTTP transport")
	deployMCPCmd.Flags().StringVar(&deployCommand, "command", "", "Command to run (overrides project config)")
	deployMCPCmd.Flags().StringSliceVar(&deployArgs, "args", []string{}, "Command arguments")
	deployMCPCmd.Flags().StringSliceVar(&deployEnv, "env", []string{}, "Environment variables (KEY=VALUE)")
	deployMCPCmd.Flags().BoolVar(&deployForce, "force", false, "Force deployment even if validation fails")
	deployMCPCmd.Flags().StringVarP(&deployFile, "file", "f", "", "Path to kmcp.yaml file (default: current directory)")
	deployMCPCmd.Flags().StringSliceVar(
		&deploySecrets,
		"secrets",
		[]string{},
		"Paths to Kubernetes secret YAML files to apply to cluster",
	)

	// Controller deployment flags
	deployControllerCmd.Flags().StringVar(
		&controllerVersion,
		"version",
		"",
		"Version of the controller to deploy (defaults to kmcp version)",
	)
	deployControllerCmd.Flags().StringVar(
		&controllerNamespace,
		"namespace",
		"kmcp-system",
		"Namespace for the KMCP controller (defaults to kmcp-system)",
	)
	deployControllerCmd.Flags().StringVar(
		&controllerRegistryConfig,
		"registry-config",
		"",
		"Path to docker registry config file",
	)
}

func runDeployMCP(_ *cobra.Command, args []string) error {
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
	yamlContent := fmt.Sprintf(
		"---\n# MCPServer deployment generated by kmcp deploy mcp\n# Project: %s\n# Framework: %s\n%s",
		projectManifest.Name,
		projectManifest.Framework,
		string(yamlData),
	)

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
		// Apply secrets first if provided
		if len(deploySecrets) > 0 {
			if err := applySecretFiles(deploySecrets); err != nil {
				return fmt.Errorf("failed to apply secrets: %w", err)
			}
		}

		// Apply MCPServer to cluster
		if err := applyToCluster(yamlContent, deploymentName); err != nil {
			return fmt.Errorf("failed to apply to cluster: %w", err)
		}
	}

	return nil
}

func runDeployController(_ *cobra.Command, _ []string) error {
	fmt.Printf("🚀 Deploying KMCP controller to cluster...\n")

	// Check if helm is available
	if err := checkHelmAvailable(); err != nil {
		return fmt.Errorf("helm is required for controller deployment: %w", err)
	}

	// Determine controller version
	version := controllerVersion
	if version == "" {
		version = getKMCPVersion()
	}

	// Validate controller version format
	if version == "" {
		return fmt.Errorf("invalid controller version: version cannot be empty")
	}

	// Determine registry config file
	registryConfig := controllerRegistryConfig
	if registryConfig == "" {
		fmt.Print("Docker registry config must be set use --registry-config\n")
	}
	if registryConfig != "" && verbose {
		fmt.Printf("Using registry config: %s\n", registryConfig)
	}

	// Build helm install command
	args := []string{
		"install", "kmcp", "oci://ghcr.io/kagent-dev/kmcp/helm/kmcp",
		"--version", version,
		"--namespace", controllerNamespace,
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
		version,
	)
	fmt.Printf(
		"💡 Check controller status with: kubectl get pods -n %s\n",
		controllerNamespace,
	)
	fmt.Printf(
		"💡 View controller logs with: kubectl logs -l app.kubernetes.io/name=kmcp-controller-manager -n %s\n",
		controllerNamespace,
	)

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

	// Parse secret files and extract references
	var secretRefs []corev1.ObjectReference
	if len(deploySecrets) > 0 {
		if verbose {
			fmt.Printf("🔍 Parsing %d secret file(s) for references...\n", len(deploySecrets))
		}
		var err error
		secretRefs, err = parseSecretFiles(deploySecrets)
		if err != nil {
			return nil, fmt.Errorf("failed to parse secret files: %w", err)
		}
		if verbose {
			fmt.Printf("✅ Found %d secret reference(s) to include in MCPServer\n", len(secretRefs))
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
				Image:      imageName,
				Port:       uint16(port),
				Cmd:        command,
				Args:       args,
				Env:        envVars,
				SecretRefs: secretRefs,
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

// applySecretFiles applies Kubernetes secret YAML files to the cluster
func applySecretFiles(secretFiles []string) error {
	fmt.Printf("🔐 Applying %d secret file(s) to cluster...\n", len(secretFiles))

	// Check if kubectl is available
	if err := checkKubectlAvailable(); err != nil {
		return fmt.Errorf("kubectl is required for secret deployment: %w", err)
	}

	for i, secretFile := range secretFiles {
		if verbose {
			fmt.Printf("Applying secret file %d/%d: %s\n", i+1, len(secretFiles), secretFile)
		}

		if err := applySecretFile(secretFile); err != nil {
			return fmt.Errorf("failed to apply secret file %s: %w", secretFile, err)
		}
	}

	if deployNamespace != "" {
		fmt.Printf("✅ Successfully applied %d secret file(s) to namespace %s\n", len(secretFiles), deployNamespace)
	} else {
		fmt.Printf("✅ Successfully applied %d secret file(s)\n", len(secretFiles))
	}
	return nil
}

// applySecretFile applies a single secret file to the cluster
func applySecretFile(secretFile string) error {
	// Validate file exists
	if _, err := os.Stat(secretFile); os.IsNotExist(err) {
		return fmt.Errorf("secret file does not exist: %s", secretFile)
	}

	if deployNamespace == "" {
		// Apply without namespace override
		return runKubectl("apply", "-f", secretFile)
	}

	// Apply with namespace override
	return applySecretWithNamespace(secretFile, deployNamespace)
}

// applySecretWithNamespace applies a secret file with a specific namespace
func applySecretWithNamespace(secretFile, namespace string) error {
	// Read and parse the secret file
	secret, err := readSecretFromFile(secretFile)
	if err != nil {
		return fmt.Errorf("failed to read secret file: %w", err)
	}

	// Update the namespace
	secret.Namespace = namespace

	// Create temporary file with updated namespace
	tmpFile, err := createTempSecretFile(secret)
	if err != nil {
		return fmt.Errorf("failed to create temp secret file: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			fmt.Printf("Warning: failed to remove temp file %s: %v\n", tmpFile.Name(), err)
		}
	}()

	// Apply the updated secret
	return runKubectl("apply", "-f", tmpFile.Name())
}

// readSecretFromFile reads and parses a secret YAML file
func readSecretFromFile(filename string) (*corev1.Secret, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var secret corev1.Secret
	if err := yaml.Unmarshal(data, &secret); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &secret, nil
}

// createTempSecretFile creates a temporary file with the secret YAML
func createTempSecretFile(secret *corev1.Secret) (*os.File, error) {
	// Marshal secret to YAML
	data, err := yaml.Marshal(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret: %w", err)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "secret-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write secret data to temp file
	if err := os.WriteFile(tmpFile.Name(), data, 0644); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close temp file: %v\n", closeErr)
		}
		if removeErr := os.Remove(tmpFile.Name()); removeErr != nil {
			fmt.Printf("Warning: failed to remove temp file: %v\n", removeErr)
		}
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	return tmpFile, nil
}

// parseSecretFiles parses Kubernetes secret YAML files and extracts their names and namespaces
func parseSecretFiles(secretFiles []string) ([]corev1.ObjectReference, error) {
	secretRefs := []corev1.ObjectReference{}
	for _, secretFile := range secretFiles {
		data, err := os.ReadFile(secretFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read secret file %s: %w", secretFile, err)
		}

		var secret corev1.Secret
		if err := yaml.Unmarshal(data, &secret); err != nil {
			return nil, fmt.Errorf("failed to parse secret file %s: %w", secretFile, err)
		}

		name := secret.GetName()
		if name == "" {
			return nil, fmt.Errorf("secret in file %s has no name", secretFile)
		}

		// always override namespace with deployment namespace to avoid the need for reference grants
		namespace := deployNamespace
		secretRefs = append(secretRefs, corev1.ObjectReference{
			Kind:      wellknown.SecretKind,
			Name:      name,
			Namespace: namespace,
		})

		if verbose {
			fmt.Printf("📋 Found secret reference: %s.%s\n", namespace, name)
		}
	}

	return secretRefs, nil
}
