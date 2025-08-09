package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kagent-dev/kmcp/pkg/cli/internal/manifest"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

// secretsCmd represents the secrets command
var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage project secrets",
	Long:  `Manage secrets for MCP server projects.`,
}

var (
	secretSourceFile string
	secretDryRun     bool
	secretDir        string
)

// syncCmd creates or updates a Kubernetes secret from an environment file
var syncCmd = &cobra.Command{
	Use:   "sync [environment]",
	Short: "Sync secrets to a Kubernetes environment from a local .env file",
	Long: `Sync secrets from a local .env file to a Kubernetes secret.

This command reads a .env file and the project's kmcp.yaml file to determine
the correct secret name and namespace for the specified environment. It then
creates or updates the Kubernetes secret directly in the cluster.

The command will look for a ".env" file in the project root by default.

Examples:
  # Sync secrets to the "staging" environment defined in kmcp.yaml
  kmcp secrets sync staging

  # Sync secrets from a custom .env file
  kmcp secrets sync staging --from-file .env.staging

  # Sync secrets from a specific project directory
  kmcp secrets sync staging --project-dir ./my-project

  # Perform a dry run to see the generated secret without applying it
  kmcp secrets sync production --dry-run
`,
	Args: cobra.ExactArgs(1),
	RunE: runSync,
}

func init() {
	addRootSubCmd(secretsCmd)

	// Add subcommands
	secretsCmd.AddCommand(syncCmd)

	// create-k8s-secret-from-env flags
	syncCmd.Flags().StringVar(&secretSourceFile, "from-file", ".env", "Source .env file to sync from")
	syncCmd.Flags().BoolVar(&secretDryRun, "dry-run", false, "Output the generated secret YAML instead of applying it")
	syncCmd.Flags().StringVarP(&secretDir, "project-dir", "d", "", "Project directory (default: current directory)")
}

func runSync(_ *cobra.Command, args []string) error {
	environment := args[0]

	// Determine project root
	projectRoot := secretDir
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
	} else {
		// Convert relative path to absolute path
		if !filepath.IsAbs(projectRoot) {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			projectRoot = filepath.Join(cwd, projectRoot)
		}
	}

	// Load manifest
	manifestManager := manifest.NewManager(projectRoot)
	if !manifestManager.Exists() {
		return fmt.Errorf("kmcp.yaml not found in %s. Please run 'kmcp init' or navigate to a valid project", projectRoot)
	}
	projectManifest, err := manifestManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load project manifest: %w", err)
	}

	// Get secret config for the environment
	secretConfig, ok := projectManifest.Secrets[environment]
	if !ok {
		return fmt.Errorf("environment '%s' not found in kmcp.yaml secrets configuration", environment)
	}

	if secretConfig.Provider != manifest.SecretProviderKubernetes {
		return fmt.Errorf(
			"the 'secrets sync' command only supports the 'kubernetes' provider, but environment '%s' uses '%s'",
			environment,
			secretConfig.Provider,
		)
	}

	// Load .env file
	envVars, err := loadEnvFile(secretSourceFile)
	if err != nil {
		return err
	}
	if len(envVars) == 0 {
		return fmt.Errorf("no variables found in source file '%s'", secretSourceFile)
	}

	// Create Kubernetes secret object
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretConfig.SecretName,
			Namespace: secretConfig.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: make(map[string][]byte),
	}

	for key, value := range envVars {
		secret.Data[key] = []byte(value)
	}

	if secretDryRun {
		yamlData, err := yaml.Marshal(secret)
		if err != nil {
			return fmt.Errorf("failed to marshal secret to YAML: %w", err)
		}
		fmt.Print(string(yamlData))
		return nil
	}

	// Apply to cluster
	return applySecretToCluster(secret)
}

func applySecretToCluster(secret *corev1.Secret) error {
	// Get kubeconfig
	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	// Check if secret exists
	_, err = clientset.CoreV1().Secrets(secret.Namespace).Get(context.TODO(), secret.Name, metav1.GetOptions{})
	if err != nil {
		// Create if it doesn't exist
		_, err = clientset.CoreV1().Secrets(secret.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}
		fmt.Printf("✅ Secret '%s' created in namespace '%s'.\n", secret.Name, secret.Namespace)
	} else {
		// Update if it exists
		_, err = clientset.CoreV1().Secrets(secret.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update secret: %w", err)
		}
		fmt.Printf("✅ Secret '%s' updated in namespace '%s'.\n", secret.Name, secret.Namespace)
	}

	return nil
}

// loadEnvFile reads environment variables from a file and returns them as a map
func loadEnvFile(filename string) (map[string]string, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("source secret file not found: %s", filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	envVars := make(map[string]string)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		if idx := strings.Index(line, "="); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			if key != "" {
				envVars[key] = value
			}
		}
	}

	return envVars, nil
}
