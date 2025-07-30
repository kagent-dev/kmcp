package cmd

import (
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
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		if err.Error() == "unexpected newline" {
			return "", nil
		}
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest release: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}
