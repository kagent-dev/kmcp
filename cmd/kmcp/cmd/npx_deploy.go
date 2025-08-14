package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var npxDeployCmd = &cobra.Command{
	Use:   "npx-deploy <server-name>",
	Short: "Deploy an MCP server using npx",
	Long: `Deploy an MCP server using npx to run Model Context Protocol servers.

This command creates an MCPServer Custom Resource Definition (CRD) that runs
an MCP server using npx. It's particularly useful for running MCP servers
that are available as npm packages.

The server name is required and will be used as the name of the MCPServer resource.
The --args flag is required and specifies the npm packages to run with npx.

Examples:
  kmcp npx-deploy my-server --args @modelcontextprotocol/server-github       # Deploy GitHub MCP server
  kmcp npx-deploy my-server --args package1 --dry-run                        # Print YAML without deploying
  kmcp npx-deploy my-server --args package1 --namespace prod                 # Deploy to prod namespace
  kmcp npx-deploy my-server --args package1 --image custom:tag               # Use custom image
  kmcp npx-deploy my-server --args package1 --port 8080                      # Use custom port
  kmcp npx-deploy my-server --args package1 --env "KEY1=value1,KEY2=value2"  # Set environment variables
  kmcp npx-deploy my-server --args package1 --secrets secret1,secret2        # Mount Kubernetes secrets`,
	Args: cobra.ExactArgs(1),
	RunE: runNpxDeploy,
}

var (
	// npx-deploy flags
	npxDeployNamespace string
	npxDeployDryRun    bool
	npxDeployImage     string
	npxDeployArgs      []string
	npxDeployPort      int
	npxDeployEnv       string
	npxDeploySecrets   []string
)

func init() {
	rootCmd.AddCommand(npxDeployCmd)

	// Get current namespace from kubeconfig
	currentNamespace, err := getCurrentNamespaceFromKubeconfig()
	if err != nil {
		// Fallback to default if unable to get current namespace
		currentNamespace = "default"
	}

	// npx-deploy flags
	npxDeployCmd.Flags().StringVarP(&npxDeployNamespace, "namespace", "n", currentNamespace, "Kubernetes namespace")
	npxDeployCmd.Flags().BoolVar(&npxDeployDryRun, "dry-run", false, "Print out the MCPServer yaml without deploying")
	npxDeployCmd.Flags().StringVar(&npxDeployImage, "image", "node:24-alpine3.21", "Docker image to use")
	npxDeployCmd.Flags().StringSliceVar(&npxDeployArgs, "args",
		[]string{}, "Arguments to pass to npx (e.g., package names)")
	npxDeployCmd.Flags().IntVar(&npxDeployPort, "port", 3000, "MCP server container port")
	npxDeployCmd.Flags().StringVar(&npxDeployEnv, "env", "",
		"Comma-separated environment variables (KEY1=value1,KEY2=value2)")
	npxDeployCmd.Flags().StringSliceVar(&npxDeploySecrets, "secrets", []string{},
		"List of Kubernetes secret names to mount to the MCP server container")

	// required flags
	_ = npxDeployCmd.MarkFlagRequired("args")
}

func runNpxDeploy(_ *cobra.Command, args []string) error {
	serverName := args[0]

	// Parse environment variables
	envMap := parseCommaSeparatedEnvVars(npxDeployEnv)

	// Convert secret names to ObjectReferences
	secretRefs := make([]corev1.ObjectReference, 0, len(npxDeploySecrets))
	for _, secretName := range npxDeploySecrets {
		secretRefs = append(secretRefs, corev1.ObjectReference{
			Kind:      "Secret",
			Name:      secretName,
			Namespace: npxDeployNamespace,
		})
	}

	// Create ServiceAccount
	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverName,
			Namespace: npxDeployNamespace,
		},
	}

	// Create MCPServer
	mcpServer := &v1alpha1.MCPServer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kagent.dev/v1alpha1",
			Kind:       "MCPServer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverName,
			Namespace: npxDeployNamespace,
		},
		Spec: v1alpha1.MCPServerSpec{
			Deployment: v1alpha1.MCPServerDeployment{
				Image:      npxDeployImage,
				Port:       uint16(npxDeployPort),
				Cmd:        "npx",
				Args:       npxDeployArgs,
				Env:        envMap,
				SecretRefs: secretRefs,
			},
			TransportType: v1alpha1.TransportTypeStdio,
		},
	}

	// Convert both resources to YAML
	serviceAccountYAML, err := yaml.Marshal(serviceAccount)
	if err != nil {
		return fmt.Errorf("failed to marshal ServiceAccount to YAML: %w", err)
	}

	mcpServerYAML, err := yaml.Marshal(mcpServer)
	if err != nil {
		return fmt.Errorf("failed to marshal MCPServer to YAML: %w", err)
	}

	if npxDeployDryRun {
		// Print YAML to stdout
		fmt.Println("---")
		fmt.Println(string(serviceAccountYAML))
		fmt.Println("---")
		fmt.Println(string(mcpServerYAML))
		return nil
	}

	// Apply both resources to cluster
	if err := applyResourcesToCluster(serviceAccountYAML, mcpServerYAML); err != nil {
		return err
	}

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

	return nil
}

func parseCommaSeparatedEnvVars(input string) map[string]string {
	if input == "" {
		return map[string]string{}
	}

	result := make(map[string]string)
	parts := strings.Split(input, ",")

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			// Split by first equals sign
			keyValue := strings.SplitN(trimmed, "=", 2)
			if len(keyValue) == 2 {
				key := strings.TrimSpace(keyValue[0])
				value := strings.TrimSpace(keyValue[1])
				// Remove quotes if present
				value = strings.Trim(value, "'\"")
				result[key] = value
			}
		}
	}

	return result
}
