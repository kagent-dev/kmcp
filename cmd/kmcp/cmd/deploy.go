package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
	"github.com/kagent-dev/kmcp/pkg/manifest"
	"github.com/kagent-dev/kmcp/pkg/wellknown"
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
)

func init() {
	rootCmd.AddCommand(deployCmd)

	// MCP deployment flags
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

func generateMCPServer(
	projectManifest *manifest.ProjectManifest,
	deploymentName,
	environment string,
) (*v1alpha1.MCPServer, error) {
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

	// Get secret reference from manifest for the specified environment
	secretRef, err := getSecretRefFromManifest(projectManifest, environment)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret reference: %w", err)
	}
	var secretRefs []corev1.ObjectReference
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
) (*corev1.ObjectReference, error) {
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
		namespace := secretProvider.Namespace
		if namespace == "" {
			return nil, fmt.Errorf("namespace not found in secret provider config for environment %s", environment)
		}

		return &corev1.ObjectReference{
			Kind:      wellknown.SecretKind,
			Name:      secretName,
			Namespace: namespace,
		}, nil
	}

	return nil, nil
}

func sanitizeLabelValue(value string) string {
	return strings.ReplaceAll(value, "+", "_")
}

func getDefaultCommand(framework string) string {
	switch framework {
	case manifest.FrameworkMCPGo:
		return "./server"
	default:
		return "python"
	}
}

func getDefaultArgs(framework string) []string {
	switch framework {
	case manifest.FrameworkFastMCPPython:
		return []string{"src/main.py"}
	case manifest.FrameworkMCPGo:
		return []string{}
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

// checkKubectlAvailable checks if kubectl is available in the system
func checkKubectlAvailable() error {
	cmd := exec.Command("kubectl", "version", "--client")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl not found or not working: %w", err)
	}
	return nil
}
