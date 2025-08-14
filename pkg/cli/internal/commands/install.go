package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Controller deployment flags
	controllerVersion   string
	controllerNamespace string
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
	addRootSubCmd(installCmd)

	// Controller deployment flags
	installCmd.Flags().StringVar(
		&controllerVersion,
		"version",
		"",
		"Version of the controller to deploy (defaults to latest)",
	)
	installCmd.Flags().StringVar(
		&controllerNamespace,
		"namespace",
		"kmcp-system",
		"Namespace for the KMCP controller (defaults to kmcp-system)",
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
		var err error
		version, err = getLatestReleaseTag("kagent-dev/kmcp")
		if err != nil {
			return fmt.Errorf("failed to get latest version: %w", err)
		}
		fmt.Printf("No version specified, using latest: %s\n", version)
	}

	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	crdHelmArgs := []string{
		"upgrade",
		"--install", "kmcp-crds", "oci://ghcr.io/kagent-dev/kmcp/helm/kmcp-crds",
		"--version", version,
		"--namespace", controllerNamespace,
		"--create-namespace",
	}

	if err := runHelm(crdHelmArgs...); err != nil {
		return fmt.Errorf("helm install failed: %w", err)
	}

	// Install controller using Helm
	controllerHelmArgs := []string{
		"upgrade",
		"--install", "kmcp", "oci://ghcr.io/kagent-dev/kmcp/helm/kmcp",
		"--version", version,
		"--namespace", controllerNamespace,
		"--create-namespace",
	}

	if err := runHelm(controllerHelmArgs...); err != nil {
		return fmt.Errorf("helm install failed: %w", err)
	}

	fmt.Printf("✅ KMCP controller deployed successfully\n")
	fmt.Printf(
		"💡 Check controller status with: kubectl get pods -n %s\n",
		controllerNamespace,
	)
	fmt.Printf(
		"💡 View controller logs with: kubectl logs -l app.kubernetes.io/name=kmcp -n %s\n",
		controllerNamespace,
	)

	return nil
}

// runHelm executes helm commands
func runHelm(args ...string) error {
	if Verbose {
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
