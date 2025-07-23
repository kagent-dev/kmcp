package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kagent.dev/kmcp/pkg/manifest"
	"kagent.dev/kmcp/pkg/secrets"
	"sigs.k8s.io/yaml"
)

// secretsCmd represents the secrets command
var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage project secrets",
	Long: `Manage secrets for MCP server projects across different environments.

This command provides secure secret management for local development and Kubernetes deployments.`,
}

// addSecretCmd adds a secret to the project
var addSecretCmd = &cobra.Command{
	Use:   "add-secret [key] [flags]",
	Short: "Add a secret to the project",
	Long: `Add a secret to the project for the specified environment.

For local environment, secrets are stored in .env.local file.
For Kubernetes environments, secrets are managed via kubectl.`,
	Args: cobra.ExactArgs(1),
	RunE: runAddSecret,
}

// listSecretsCmd lists secrets for an environment
var listSecretsCmd = &cobra.Command{
	Use:   "list-secrets [flags]",
	Short: "List secrets for an environment",
	Long: `List available secrets for the specified environment.

This shows the secret keys (not values) that are configured for the environment.`,
	RunE: runListSecrets,
}

// generateK8sSecretsCmd generates Kubernetes secret manifests
var generateK8sSecretsCmd = &cobra.Command{
	Use:   "generate-k8s-secrets [flags]",
	Short: "Generate Kubernetes secret manifests",
	Long: `Generate Kubernetes secret manifests for the specified environment.

This creates YAML files that can be applied to a Kubernetes cluster.`,
	RunE: runGenerateK8sSecrets,
}

// validateSecretsCmd validates secret configuration
var validateSecretsCmd = &cobra.Command{
	Use:   "validate-secrets [flags]",
	Short: "Validate secret configuration",
	Long:  `Validate that all required secrets are configured and accessible for the specified environment.`,
	RunE:  runValidateSecrets,
}

// createSecretFromEnvCmd creates a Kubernetes secret from an environment file
var createSecretFromEnvCmd = &cobra.Command{
	Use:   "create-secret-from-env [env-file] [flags]",
	Short: "Create a Kubernetes secret from an environment file",
	Long: `Create a Kubernetes secret manifest from an environment file.

This command reads a .env file and generates a Kubernetes Secret YAML manifest.
The environment file should contain key=value pairs, one per line.
The output filename will match the input filename with .yaml extension.

Examples:
  kmcp secrets create-secret-from-env .env.local
  kmcp secrets create-secret-from-env .env.production --name my-app-secrets --namespace production
  kmcp secrets create-secret-from-env .env.staging --output-dir secrets/
  kmcp secrets create-secret-from-env /your-mcp-server/env.local --name secret --output-dir your-mcp-server/secrets/`,
	Args: cobra.ExactArgs(1),
	RunE: runCreateSecretFromEnv,
}

func init() {
	rootCmd.AddCommand(secretsCmd)

	// Add subcommands
	secretsCmd.AddCommand(addSecretCmd)
	secretsCmd.AddCommand(listSecretsCmd)
	secretsCmd.AddCommand(generateK8sSecretsCmd)
	secretsCmd.AddCommand(validateSecretsCmd)
	secretsCmd.AddCommand(createSecretFromEnvCmd)

	// add-secret flags
	addSecretCmd.Flags().StringP("environment", "e", "local", "Environment to add secret to (local, staging, production)")
	addSecretCmd.Flags().StringP("value", "v", "", "Secret value (will prompt if not provided)")
	addSecretCmd.Flags().Bool("from-stdin", false, "Read secret value from stdin")

	// list-secrets flags
	listSecretsCmd.Flags().StringP("environment", "e", "local", "Environment to list secrets for")

	// generate-k8s-secrets flags
	generateK8sSecretsCmd.Flags().StringP("environment", "e", "staging", "Environment to generate secrets for")
	generateK8sSecretsCmd.Flags().StringP("output", "o", "", "Output file (default: secrets/{env}-secrets.yaml)")

	// validate-secrets flags
	validateSecretsCmd.Flags().StringP("environment", "e", "local", "Environment to validate")
	validateSecretsCmd.Flags().Bool("scan-responses", false, "Scan for potential secret leakage in responses")

	// create-secret-from-env flags
	createSecretFromEnvCmd.Flags().StringP("name", "n", "", "Kubernetes secret name (default: derived from env file name)")
	createSecretFromEnvCmd.Flags().StringP("namespace", "s", "default", "Kubernetes namespace")
	createSecretFromEnvCmd.Flags().StringP("output-dir", "o", "", "Output directory (default: stdout)")

}

func runAddSecret(cmd *cobra.Command, args []string) error {
	key := args[0]
	environment, _ := cmd.Flags().GetString("environment")
	value, _ := cmd.Flags().GetString("value")
	fromStdin, _ := cmd.Flags().GetBool("from-stdin")

	// Load project manifest
	manifestManager := manifest.NewManager(".")
	if !manifestManager.Exists() {
		return fmt.Errorf("kmcp.yaml not found. Run 'kmcp init' first")
	}

	projectManifest, err := manifestManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load project manifest: %w", err)
	}

	// Get secret configuration for environment
	secretConfig, err := manifestManager.GetSecretConfig(projectManifest, environment)
	if err != nil {
		return fmt.Errorf("failed to get secret config: %w", err)
	}

	// Get secret value if not provided
	if value == "" && !fromStdin {
		fmt.Printf("Enter value for secret '%s': ", key)
		if _, err := fmt.Scanln(&value); err != nil {
			return fmt.Errorf("failed to read secret value: %w", err)
		}
	} else if fromStdin {
		var input strings.Builder
		buffer := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buffer)
			if n > 0 {
				input.Write(buffer[:n])
			}
			if err != nil {
				break
			}
		}
		value = strings.TrimSpace(input.String())
	}

	if value == "" {
		return fmt.Errorf("secret value cannot be empty")
	}

	// Create secret manager and add secret
	secretManager, err := secrets.NewManager(environment, secretConfig)
	if err != nil {
		return fmt.Errorf("failed to create secret manager: %w", err)
	}

	if err := secretManager.Set(key, value); err != nil {
		return fmt.Errorf("failed to set secret: %w", err)
	}

	fmt.Printf("✅ Secret '%s' added to %s environment\n", key, environment)
	return nil
}

func runListSecrets(cmd *cobra.Command, _ []string) error {
	environment, _ := cmd.Flags().GetString("environment")

	// Load project manifest
	manifestManager := manifest.NewManager(".")
	if !manifestManager.Exists() {
		return fmt.Errorf("kmcp.yaml not found. Run 'kmcp init' first")
	}

	projectManifest, err := manifestManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load project manifest: %w", err)
	}

	// Get secret configuration for environment
	secretConfig, err := manifestManager.GetSecretConfig(projectManifest, environment)
	if err != nil {
		return fmt.Errorf("failed to get secret config: %w", err)
	}

	// Create secret manager and list secrets
	secretManager, err := secrets.NewManager(environment, secretConfig)
	if err != nil {
		return fmt.Errorf("failed to create secret manager: %w", err)
	}

	keys, err := secretManager.ListKeys()
	if err != nil {
		return fmt.Errorf("failed to list secrets: %w", err)
	}

	if len(keys) == 0 {
		fmt.Printf("No secrets found in %s environment\n", environment)
		return nil
	}

	fmt.Printf("Secrets in %s environment:\n", environment)
	for _, key := range keys {
		exists := secretManager.Exists(key)
		status := "✅"
		if !exists {
			status = "❌"
		}
		fmt.Printf("  %s %s\n", status, key)
	}

	return nil
}

func runGenerateK8sSecrets(cmd *cobra.Command, _ []string) error {
	environment, _ := cmd.Flags().GetString("environment")
	output, _ := cmd.Flags().GetString("output")

	// Load project manifest
	manifestManager := manifest.NewManager(".")
	if !manifestManager.Exists() {
		return fmt.Errorf("kmcp.yaml not found. Run 'kmcp init' first")
	}

	projectManifest, err := manifestManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load project manifest: %w", err)
	}

	// Get secret configuration for environment
	secretConfig, err := manifestManager.GetSecretConfig(projectManifest, environment)
	if err != nil {
		return fmt.Errorf("failed to get secret config: %w", err)
	}

	if secretConfig.Provider != manifest.SecretProviderKubernetes {
		return fmt.Errorf("environment %s is not configured for Kubernetes secrets", environment)
	}

	// Set default output file
	if output == "" {
		if err := os.MkdirAll("secrets", 0755); err != nil {
			return fmt.Errorf("failed to create secrets directory: %w", err)
		}
		output = fmt.Sprintf("secrets/%s-secrets.yaml", environment)
	}

	// For Kubernetes secrets, we need to create a template
	// In a real implementation, this would read from a local source
	secretData := map[string]string{
		"EXAMPLE_API_KEY": "your-api-key-here",
		"DATABASE_URL":    "postgresql://user:pass@host:5432/db",
	}

	// Create Kubernetes secret YAML
	yamlData, err := createKubernetesSecretYAML(secretData, secretConfig.SecretName, secretConfig.Namespace)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes secret: %w", err)
	}

	// Write to file
	if err := os.WriteFile(output, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write secret file: %w", err)
	}

	fmt.Printf("✅ Kubernetes secret manifest generated: %s\n", output)
	fmt.Printf("💡 Edit the file to set actual secret values, then apply with: kubectl apply -f %s\n", output)
	return nil
}

func runValidateSecrets(cmd *cobra.Command, _ []string) error {
	environment, _ := cmd.Flags().GetString("environment")
	scanResponses, _ := cmd.Flags().GetBool("scan-responses")

	// Load project manifest
	manifestManager := manifest.NewManager(".")
	if !manifestManager.Exists() {
		return fmt.Errorf("kmcp.yaml not found. Run 'kmcp init' first")
	}

	projectManifest, err := manifestManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load project manifest: %w", err)
	}

	// Get secret configuration for environment
	secretConfig, err := manifestManager.GetSecretConfig(projectManifest, environment)
	if err != nil {
		return fmt.Errorf("failed to get secret config: %w", err)
	}

	// Create secret manager
	secretManager, err := secrets.NewManager(environment, secretConfig)
	if err != nil {
		return fmt.Errorf("failed to create secret manager: %w", err)
	}

	// Validate secret configuration
	fmt.Printf("🔍 Validating secret configuration for %s environment...\n", environment)

	// Check if secrets are accessible
	keys, err := secretManager.ListKeys()
	if err != nil {
		fmt.Printf("❌ Failed to access secrets: %v\n", err)
		return err
	}

	fmt.Printf("✅ Secret configuration valid\n")
	fmt.Printf("✅ Found %d secrets\n", len(keys))

	// Check for required secrets (this would be customizable)
	requiredSecrets := []string{"API_KEY", "DATABASE_URL"}
	for _, required := range requiredSecrets {
		if secretManager.Exists(required) {
			fmt.Printf("✅ Required secret '%s' present\n", required)
		} else {
			fmt.Printf("⚠️  Required secret '%s' missing\n", required)
		}
	}

	// Check for unused secrets
	for _, key := range keys {
		found := false
		for _, required := range requiredSecrets {
			if key == required {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("⚠️  Secret '%s' not in required list\n", key)
		}
	}

	// Scan for potential secret leakage if requested
	if scanResponses {
		fmt.Printf("🔍 Scanning for potential secret leakage patterns...\n")

		// Example: test sanitization
		testData := map[string]interface{}{
			"api_response": "Bearer sk-1234567890abcdef",
			"config": map[string]string{
				"database_url": "postgresql://user:secret123@host:5432/db",
			},
		}

		sanitized := secretManager.SanitizeForMCP(testData)

		// Simple check to see if sanitization is working
		sanitizedBytes, _ := yaml.Marshal(sanitized)
		if strings.Contains(string(sanitizedBytes), "secret123") ||
			strings.Contains(string(sanitizedBytes), "sk-1234567890abcdef") {
			fmt.Printf("❌ Potential secret leak detected in test data\n")
		} else {
			fmt.Printf("✅ Secret sanitization working correctly\n")
		}
	}

	return nil
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

// createKubernetesSecretForEnvVars creates a Kubernetes secret with a .env file
// from the raw file content
func createKubernetesSecretForEnvVars(envFileContent []byte, secretName, namespace string) ([]byte, error) {
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

	// Store the raw .env file content under the ".env" key
	secret.Data[".env"] = envFileContent

	// Marshal to YAML
	yamlData, err := yaml.Marshal(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret to YAML: %w", err)
	}

	return yamlData, nil
}

func runCreateSecretFromEnv(cmd *cobra.Command, args []string) error {
	envFile := args[0]

	secretName, _ := cmd.Flags().GetString("name")
	namespace, _ := cmd.Flags().GetString("namespace")
	outputDir, _ := cmd.Flags().GetString("output-dir")

	// Validate env file exists
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf("environment file not found: %s", envFile)
	}

	// Read the raw file content
	fileContent, err := os.ReadFile(envFile)
	if err != nil {
		return fmt.Errorf("failed to read environment file: %w", err)
	}

	if len(fileContent) == 0 {
		return fmt.Errorf("environment file is empty: %s", envFile)
	}

	// Generate secret name if not provided
	if secretName == "" {
		secretName = generateSecretNameFromFile(envFile)
	}

	// Generate secret name if not provided
	if secretName == "" {
		secretName = generateSecretNameFromFile(envFile)
	}

	// Create Kubernetes secret YAML with .env file
	yamlData, err := createKubernetesSecretForEnvVars(fileContent, secretName, namespace)
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

// loadEnvFile reads environment variables from a file
func loadEnvFile(filename string) (map[string]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	envVars := make(map[string]string)
	lines := strings.Split(string(data), "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format on line %d: %s", i+1, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		if key == "" {
			return nil, fmt.Errorf("empty key on line %d", i+1)
		}

		envVars[key] = value
	}

	return envVars, nil
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
