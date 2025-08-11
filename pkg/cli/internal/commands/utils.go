package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

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

	if strings.HasPrefix(release.TagName, "v") {
		release.TagName = strings.TrimPrefix(release.TagName, "v")
	}

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
