package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
	"github.com/kagent-dev/kmcp/pkg/cli/internal/manifest"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	transportHTTP  = "http"
	transportStdio = "stdio"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
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
  kmcp deploy                          # Deploy with project name to cluster
  kmcp deploy my-server                # Deploy with custom name
  kmcp deploy --namespace staging      # Deploy to staging namespace
  kmcp deploy --dry-run                # Generate manifest without applying to cluster
  kmcp deploy --image custom:tag       # Use custom image
  kmcp deploy --transport http         # Use HTTP transport
  kmcp deploy --output deploy.yaml     # Save to file
  kmcp deploy --file /path/to/kmcp.yaml # Use custom kmcp.yaml file
  kmcp deploy --environment staging    # Target environment for deployment (e.g., staging, production)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDeployMCP,
}

var (
	// MCP deployment flags
	deployNamespace   string
	deployDryRun      bool
	deployOutput      string
	deployImage       string
	deployTransport   string
	deployPort        int
	deployTargetPort  int
	deployCommand     string
	deployArgs        []string
	deployEnv         []string
	deployForce       bool
	deployFile        string
	deployEnvironment string
	deployNoInspector bool
)

func init() {
	addRootSubCmd(deployCmd)

	// Get current namespace from kubeconfig
	currentNamespace, err := getCurrentNamespaceFromKubeconfig()
	if err != nil {
		// Fallback to default if unable to get current namespace
		currentNamespace = "default"
	}

	// MCP deployment flags
	deployCmd.Flags().StringVarP(&deployNamespace, "namespace", "n", currentNamespace, "Kubernetes namespace")
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
	deployCmd.Flags().BoolVar(&deployNoInspector, "no-inspector", false, "Do not start the MCP inspector after deployment")
	deployCmd.Flags().StringVar(
		&deployEnvironment,
		"environment",
		"staging",
		"Target environment for deployment (e.g., staging, production)",
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

	// Validate transport configuration
	if deployTransport == transportHTTP {
		if deployTargetPort == 0 {
			return fmt.Errorf("--target-port is required when --transport is set to 'http'")
		}

		// Determine the deployment port to compare against
		deploymentPort := deployPort
		if deploymentPort == 0 {
			deploymentPort = 3000 // Default port
		}

		if deployTargetPort == deploymentPort {
			return fmt.Errorf("--target-port (%d) must be different from deployment port (%d) when using HTTP transport", deployTargetPort, deploymentPort)
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
	mcpServer, err := generateMCPServer(projectManifest, deploymentName, deployEnvironment)
	if err != nil {
		return fmt.Errorf("failed to generate MCPServer: %w", err)
	}

	// Set namespace
	mcpServer.Namespace = deployNamespace

	if Verbose {
		fmt.Printf("Generated MCPServer: %s/%s\n", mcpServer.Namespace, mcpServer.Name)
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(mcpServer)
	if err != nil {
		return fmt.Errorf("failed to marshal MCPServer to YAML: %w", err)
	}

	// Add YAML document separator and standard header
	yamlContent := fmt.Sprintf(
		"---\n# MCPServer deployment generated by kmcp deploy\n# Project: %s\n# Framework: %s\n%s",
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
		// Apply MCPServer to cluster
		if err := applyToCluster(projectDir, yamlContent, mcpServer); err != nil {
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

func generateMCPServer(
	projectManifest *manifest.ProjectManifest,
	deploymentName,
	environment string,
) (*v1alpha1.MCPServer, error) {
	// Determine image name
	imageName := deployImage
	if imageName == "" {
		// Generate default image name
		imageName = fmt.Sprintf("%s:%s",
			strings.ToLower(strings.ReplaceAll(projectManifest.Name, "_", "-")),
			projectManifest.Version,
		)
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
		port = 3000 // Default port
	}

	// Determine command and args
	command := deployCommand
	args := deployArgs
	if command == "" {
		// Set default command based on framework
		command = getDefaultCommand(projectManifest.Framework)
		if len(args) == 0 {
			args = getDefaultArgs(projectManifest.Framework, deployTargetPort)
		}
	}

	// Parse environment variables
	envVars := parseEnvVars(deployEnv)

	// Get secret reference from manifest for the specified environment
	secretRef, err := getSecretRefFromManifest(projectManifest, environment)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret reference: %w", err)
	}
	var secretRefs []corev1.LocalObjectReference
	if secretRef != nil {
		secretRefs = append(secretRefs, *secretRef)
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
				"kmcp.dev/version":             sanitizeLabelValue(projectManifest.Version),
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

func getSecretRefFromManifest(
	projectManifest *manifest.ProjectManifest,
	environment string,
) (*corev1.LocalObjectReference, error) {
	if environment == "" {
		return nil, nil // No environment specified
	}

	secretProvider, ok := projectManifest.Secrets[environment]
	if !ok {
		return nil, fmt.Errorf("environment '%s' not found in secrets config", environment)
	}

	if secretProvider.Provider == manifest.SecretProviderKubernetes && secretProvider.Enabled {
		secretName := secretProvider.SecretName
		if secretName == "" {
			return nil, fmt.Errorf("secretName not found in secret provider config for environment %s", environment)
		}

		return &corev1.LocalObjectReference{
			Name: secretName,
		}, nil
	}

	return nil, nil
}

func sanitizeLabelValue(value string) string {
	return strings.ReplaceAll(value, "+", "_")
}

func getDefaultCommand(framework string) string {
	switch framework {
	case manifest.FrameworkFastMCPPython:
		return "python"
	case manifest.FrameworkMCPGo:
		return "./server"
	case manifest.FrameworkTypeScript:
		return "node"
	default:
		return "python"
	}
}

func getDefaultArgs(framework string, targetPort int) []string {
	switch framework {
	case manifest.FrameworkFastMCPPython:
		if deployTransport == transportHTTP {
			return []string{"src/main.py", "--transport", "sse", "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", targetPort)}
		}
		return []string{"src/main.py"}
	case manifest.FrameworkMCPGo:
		return []string{}
	case manifest.FrameworkTypeScript:
		return []string{"dist/index.js"}
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

func applyToCluster(projectDir, yamlContent string, mcpServer *v1alpha1.MCPServer) error {
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
	err = runKubectl("apply", "-f", tmpFile.Name())
	if err != nil {
		// Check for CRD not found error
		if strings.Contains(err.Error(), "no matches for kind") {
			return fmt.Errorf("MCPServer CRD not found. Please run 'kmcp install' first")
		}
		return fmt.Errorf("kubectl apply failed: %w", err)
	}

	fmt.Printf("✅ MCPServer '%s' applied successfully\n", mcpServer.Name)

	// Wait for the deployment to be ready
	fmt.Printf("⌛ Waiting for deployment '%s' to be ready...\n", mcpServer.Name)
	if err := waitForDeployment(mcpServer.Name, mcpServer.Namespace, 2*time.Minute); err != nil {
		return fmt.Errorf("deployment not ready: %w", err)
	}

	fmt.Printf("✅ Deployment '%s' is ready.\n", mcpServer.Name)
	fmt.Printf("💡 Check status with: kubectl get mcpserver %s -n %s\n", mcpServer.Name, mcpServer.Namespace)
	fmt.Printf("💡 View logs with: kubectl logs -l app.kubernetes.io/name=%s -n %s\n", mcpServer.Name, mcpServer.Namespace)
	if mcpServer.Spec.Deployment.Port != 0 {
		fmt.Printf("💡 Port-forward to the service with: "+
			"kubectl port-forward service/%s %d:%d -n %s\n",
			mcpServer.Name, mcpServer.Spec.Deployment.Port,
			mcpServer.Spec.Deployment.Port, mcpServer.Namespace)
	}

	var configPath string
	if !deployNoInspector {
		// Create inspector config
		port := uint16(3000) // default port
		if mcpServer.Spec.Deployment.Port != 0 {
			port = mcpServer.Spec.Deployment.Port
		}
		serverConfig := map[string]interface{}{
			"type": "streamable-http",
			"url":  fmt.Sprintf("http://localhost:%d/mcp", port),
		}
		configPath = filepath.Join(projectDir, "mcp-server-config.json")
		if err := createMCPInspectorConfig(mcpServer.Name, serverConfig, configPath); err != nil {
			return fmt.Errorf("failed to create inspector config: %w", err)
		}

		if err := runInspector(mcpServer, configPath, projectDir); err != nil {
			return fmt.Errorf("failed to run inspector: %w", err)
		}
	}
	if err := os.Remove(tmpFile.Name()); err != nil {
		fmt.Printf("failed to remove temp file: %v\n", err)
	}
	return nil
}

func runInspector(mcpServer *v1alpha1.MCPServer, configPath string, projectDir string) error {
	// Check if npx is installed
	if err := checkNpxInstalled(); err != nil {
		return err
	}

	// Start port forwarding in the background
	portForwardCmd, err := runPortForward(mcpServer)
	if err != nil {
		return err
	}
	defer func() {
		if portForwardCmd != nil && portForwardCmd.Process != nil {
			if err := portForwardCmd.Process.Kill(); err != nil {
				fmt.Printf("failed to kill port-forward process: %v\n", err)
			}
		}
	}()

	// Run the inspector
	return runMCPInspector(configPath, mcpServer.Name, projectDir)
}

func runPortForward(mcpServer *v1alpha1.MCPServer) (*exec.Cmd, error) {
	remotePort := mcpServer.Spec.Deployment.Port
	if remotePort == 0 {
		remotePort = 3000 // Default port
	}
	localPort := 3000
	portMapping := fmt.Sprintf("%d:%d", localPort, remotePort)
	args := []string{
		"port-forward",
		"service/" + mcpServer.Name,
		portMapping,
		"-n", mcpServer.Namespace,
	}
	cmd := exec.Command("kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start port-forward: %w", err)
	}
	return cmd, nil
}

func waitForDeployment(name, namespace string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{
		"rollout", "status", "deployment", name,
		"-n", namespace,
		"--timeout", timeout.String(),
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	if Verbose {
		fmt.Printf("Running: kubectl %s\n", strings.Join(args, " "))
	}
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	// sleep 1 second just to allow controller to create the deployment
	time.Sleep(1 * time.Second)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timed out waiting for deployment to be ready")
		}
		return fmt.Errorf("`kubectl rollout status` failed: %w\n%s", err, stderr.String())
	}
	return nil
}

func runKubectl(args ...string) error {
	if Verbose {
		fmt.Printf("Running: kubectl %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("kubectl", args...)
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("`kubectl %s` failed: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}

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
