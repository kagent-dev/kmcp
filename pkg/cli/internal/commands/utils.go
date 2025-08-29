package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"k8s.io/client-go/tools/clientcmd"
)

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func promptForInput(promptText string) (string, error) {
	fmt.Print(promptText)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// getCurrentNamespaceFromKubeconfig returns the current namespace from the kubeconfig
func getCurrentNamespaceFromKubeconfig() (string, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	namespace, _, err := config.Namespace()
	if err != nil {
		return "", err
	}
	return namespace, nil
}

func getLatestReleaseTag(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest release: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	release.TagName = strings.TrimPrefix(release.TagName, "v")

	return release.TagName, nil
}

func getCurrentKindClusterName() (string, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	rawConfig, err := config.RawConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get raw kubeconfig: %w", err)
	}

	currentContext, ok := rawConfig.Contexts[rawConfig.CurrentContext]
	if !ok {
		return "", fmt.Errorf("current context %q not found in kubeconfig", rawConfig.CurrentContext)
	}

	const kindPrefix = "kind-"
	if strings.HasPrefix(currentContext.Cluster, kindPrefix) {
		return strings.TrimPrefix(currentContext.Cluster, kindPrefix), nil
	}

	return "", fmt.Errorf("current cluster %q is not a kind cluster", currentContext.Cluster)
}

// applyResourcesToCluster applies multiple YAML resources to the Kubernetes cluster
func applyResourcesToCluster(yamls ...[]byte) error {
	fmt.Printf("🚀 Applying resources to cluster...\n")

	// Check if kubectl is available
	if err := checkKubectlAvailable(); err != nil {
		return fmt.Errorf("kubectl is required for cluster deployment: %w", err)
	}

	// Create temporary file for kubectl apply
	tmpFile, err := os.CreateTemp("", "resources-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			fmt.Printf("failed to remove temp file: %v\n", err)
		}
	}()

	// Combine all YAML resources with separators
	var combinedYAML []byte
	for i, yaml := range yamls {
		if i > 0 {
			combinedYAML = append(combinedYAML, []byte("\n---\n")...)
		}
		combinedYAML = append(combinedYAML, yaml...)
	}

	if _, err := tmpFile.Write(combinedYAML); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
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

	fmt.Printf("✅ Resources applied successfully\n")
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

// runKubectl runs a kubectl command with the given arguments
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

// waitForDeployment waits for a Kubernetes deployment to be ready
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
