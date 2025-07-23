package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// secretsCmd represents the secrets command
var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage project secrets",
	Long: `Manage secrets for MCP server projects.

This command currently only supports creating Kubernetes secrets from environment files.`,
}

// createK8SecretFromEnvCmd creates a Kubernetes secret from an environment file
var createK8SecretFromEnvCmd = &cobra.Command{
	Use:   "create-k8s-secret-from-env [env-file] [flags]",
	Short: "Create a Kubernetes secret from an environment file",
	Long: `Create a Kubernetes secret manifest from an environment file.

This command reads a .env file and generates a Kubernetes Secret YAML manifest.
The environment file should contain key=value pairs, one per line.
The output filename will match the input filename with .yaml extension.

Examples:
  kmcp secrets create-k8s-secret-from-env .env.local
  kmcp secrets create-k8s-secret-from-env .env.production --name my-app-secrets --namespace production
  kmcp secrets create-k8s-secret-from-env .env.staging --output-dir secrets/
  kmcp secrets create-k8s-secret-from-env /your-mcp-server/env.local --name secret --output-dir your-mcp-server/secrets/`,
	Args: cobra.ExactArgs(1),
	RunE: runCreateK8SecretFromEnv,
}

func init() {
	rootCmd.AddCommand(secretsCmd)

	// Add subcommands
	secretsCmd.AddCommand(createK8SecretFromEnvCmd)

	// create-k8s-secret-from-env flags
	createK8SecretFromEnvCmd.Flags().StringP("name", "n", "", "Kubernetes secret name (default: derived from env file name)")
	createK8SecretFromEnvCmd.Flags().StringP("namespace", "s", "default", "Kubernetes namespace")
	createK8SecretFromEnvCmd.Flags().StringP("output-dir", "o", "", "Output directory (default: stdout)")
}

// createKubernetesSecretYAML creates a Kubernetes Secret YAML from secret data
func createKubernetesSecretYAML(secretData map[string]string, secretName, namespace string) ([]byte, error) {
	// Create Kubernetes secret
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: make(map[string][]byte),
	}

	// Convert string data to byte data
	for key, value := range secretData {
		secret.Data[key] = []byte(value)
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret to YAML: %w", err)
	}

	return yamlData, nil
}

// loadEnvFile reads environment variables from a file and returns them as a map
func loadEnvFile(filename string) (map[string]string, error) {
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

func runCreateK8SecretFromEnv(cmd *cobra.Command, args []string) error {
	envFile := args[0]

	secretName, _ := cmd.Flags().GetString("name")
	namespace, _ := cmd.Flags().GetString("namespace")
	outputDir, _ := cmd.Flags().GetString("output-dir")

	// Validate env file exists
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf("environment file not found: %s", envFile)
	}

	// Parse the .env file into key-value pairs
	envVars, err := loadEnvFile(envFile)
	if err != nil {
		return fmt.Errorf("failed to parse environment file: %w", err)
	}

	if len(envVars) == 0 {
		return fmt.Errorf("environment file contains no valid key-value pairs: %s", envFile)
	}

	// Generate secret name if not provided
	if secretName == "" {
		secretName = generateSecretNameFromFile(envFile)
	}

	// Create Kubernetes secret YAML with individual key-value pairs
	yamlData, err := createKubernetesSecretYAML(envVars, secretName, namespace)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes secret: %w", err)
	}

	// Handle output
	if outputDir != "" {
		// Create output directory if it doesn't exist
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Generate output filename based on input filename
		inputFileName := filepath.Base(envFile)
		outputFileName := strings.TrimSuffix(inputFileName, filepath.Ext(inputFileName)) + ".yaml"
		outputPath := filepath.Join(outputDir, outputFileName)

		// Write to file
		if err := os.WriteFile(outputPath, yamlData, 0644); err != nil {
			return fmt.Errorf("failed to write secret file: %w", err)
		}
		fmt.Printf("✅ Kubernetes secret manifest written to: %s\n", outputPath)
	} else {
		// Print to stdout
		fmt.Print(string(yamlData))
	}

	return nil
}

// generateSecretNameFromFile generates a secret name from the environment file name
func generateSecretNameFromFile(filename string) string {
	// Get the base name without extension
	baseName := filepath.Base(filename)
	name := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Remove common prefixes/suffixes
	name = strings.TrimPrefix(name, ".")
	name = strings.TrimPrefix(name, "env")
	name = strings.TrimPrefix(name, "-")
	name = strings.TrimSuffix(name, "-")

	// Convert to kebab-case and add suffix
	if name == "" {
		name = "app"
	}

	// Replace underscores and dots with hyphens
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")

	// Ensure it's a valid Kubernetes name
	name = strings.ToLower(name)

	return fmt.Sprintf("%s-secrets", name)
}
