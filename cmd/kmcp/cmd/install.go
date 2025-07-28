package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Controller deployment flags
	controllerVersion        string
	controllerNamespace      string
	controllerRegistryConfig string
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the KMCP controller on a Kubernetes cluster",
	Long: `Install the KMCP controller and its required Custom Resource Definitions (CRDs)
on a Kubernetes cluster.

This command should be run once per cluster to set up the necessary infrastructure
for deploying MCP servers.

It will install the following resources:
- MCPServer Custom Resource Definition
- ClusterRole and ClusterRoleBinding for RBAC
- The KMCP controller Deployment`,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)

	// Controller deployment flags
	installCmd.Flags().StringVar(
		&controllerVersion,
		"version",
		"",
		"Version of the controller to deploy (defaults to kmcp version)",
	)
	installCmd.Flags().StringVar(
		&controllerNamespace,
		"namespace",
		"kmcp-system",
		"Namespace for the KMCP controller (defaults to kmcp-system)",
	)
	installCmd.Flags().StringVar(
		&controllerRegistryConfig,
		"registry-config",
		"",
		"Path to docker registry config file",
	)
}

func runInstall(_ *cobra.Command, _ []string) error {
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

// checkHelmAvailable checks if helm is available in the system
func checkHelmAvailable() error {
	cmd := exec.Command("helm", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm not found or not working: %w", err)
	}
	return nil
}

// getKMCPVersion returns the current kmcp version
func getKMCPVersion() string {
	return Version
}
