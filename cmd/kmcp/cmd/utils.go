package cmd

import (
	"fmt"
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
