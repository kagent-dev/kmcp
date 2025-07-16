package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const ManifestFileName = "kmcp.yaml"

// Manager handles project manifest operations
type Manager struct {
	projectRoot string
}

// NewManager creates a new manifest manager
func NewManager(projectRoot string) *Manager {
	return &Manager{
		projectRoot: projectRoot,
	}
}

// Load reads and parses the kmcp.yaml file
func (m *Manager) Load() (*ProjectManifest, error) {
	manifestPath := filepath.Join(m.projectRoot, ManifestFileName)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("kmcp.yaml not found in %s", m.projectRoot)
		}
		return nil, fmt.Errorf("failed to read kmcp.yaml: %w", err)
	}

	var manifest ProjectManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse kmcp.yaml: %w", err)
	}

	// Validate the manifest
	if err := m.Validate(&manifest); err != nil {
		return nil, fmt.Errorf("invalid kmcp.yaml: %w", err)
	}

	return &manifest, nil
}

// Save writes the manifest to kmcp.yaml
func (m *Manager) Save(manifest *ProjectManifest) error {
	// Update timestamp
	manifest.UpdatedAt = time.Now()

	// Validate before saving
	if err := m.Validate(manifest); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestPath := filepath.Join(m.projectRoot, ManifestFileName)
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write kmcp.yaml: %w", err)
	}

	return nil
}

// Create generates a new manifest with default values
func (m *Manager) Create(name, framework string) (*ProjectManifest, error) {
	if name == "" {
		return nil, fmt.Errorf("project name is required")
	}

	if !isValidFramework(framework) {
		return nil, fmt.Errorf("unsupported framework: %s", framework)
	}

	manifest := &ProjectManifest{
		Name:        name,
		Framework:   framework,
		Version:     "1.0.0",
		Description: fmt.Sprintf("MCP server built with %s", framework),
		Tools:       make(map[string]ToolConfig),
		Resources:   make(map[string]ResourceConfig),
		Secrets: SecretsConfig{
			Local: SecretProviderConfig{
				Provider: SecretProviderEnv,
				Source:   ".env.local",
			},
			Staging: SecretProviderConfig{
				Provider:   SecretProviderKubernetes,
				SecretName: fmt.Sprintf("%s-secrets-staging", name),
				Namespace:  "staging",
			},
			Production: SecretProviderConfig{
				Provider:   SecretProviderKubernetes,
				SecretName: fmt.Sprintf("%s-secrets", name),
				Namespace:  "default",
			},
		},
		Dependencies: DependencyConfig{
			AutoManage: true,
		},
		Build: BuildConfig{
			Docker: DockerConfig{
				Port: 3000,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return manifest, nil
}

// Exists checks if a kmcp.yaml file exists in the project root
func (m *Manager) Exists() bool {
	manifestPath := filepath.Join(m.projectRoot, ManifestFileName)
	_, err := os.Stat(manifestPath)
	return err == nil
}

// Validate checks if the manifest is valid
func (m *Manager) Validate(manifest *ProjectManifest) error {
	// Basic validation
	if manifest.Name == "" {
		return fmt.Errorf("project name is required")
	}

	if manifest.Framework == "" {
		return fmt.Errorf("framework is required")
	}

	if !isValidFramework(manifest.Framework) {
		return fmt.Errorf("unsupported framework: %s", manifest.Framework)
	}

	// Validate tools
	for toolName, tool := range manifest.Tools {
		if err := m.validateTool(toolName, tool); err != nil {
			return fmt.Errorf("invalid tool %s: %w", toolName, err)
		}
	}

	// Validate resources
	for resourceName, resource := range manifest.Resources {
		if err := m.validateResource(resourceName, resource); err != nil {
			return fmt.Errorf("invalid resource %s: %w", resourceName, err)
		}
	}

	// Validate secrets configuration
	if err := m.validateSecrets(manifest.Secrets); err != nil {
		return fmt.Errorf("invalid secrets configuration: %w", err)
	}

	return nil
}

// AddTool adds a new tool to the manifest
func (m *Manager) AddTool(manifest *ProjectManifest, name string, config ToolConfig) error {
	if name == "" {
		return fmt.Errorf("tool name is required")
	}

	if err := m.validateTool(name, config); err != nil {
		return err
	}

	if manifest.Tools == nil {
		manifest.Tools = make(map[string]ToolConfig)
	}

	manifest.Tools[name] = config
	return nil
}

// RemoveTool removes a tool from the manifest
func (m *Manager) RemoveTool(manifest *ProjectManifest, name string) error {
	if manifest.Tools == nil {
		return fmt.Errorf("tool %s not found", name)
	}

	if _, exists := manifest.Tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}

	delete(manifest.Tools, name)
	return nil
}

// GetSecretConfig returns the secret configuration for a specific environment
func (m *Manager) GetSecretConfig(manifest *ProjectManifest, environment string) (*SecretProviderConfig, error) {
	switch environment {
	case "local":
		return &manifest.Secrets.Local, nil
	case "staging":
		return &manifest.Secrets.Staging, nil
	case "production":
		return &manifest.Secrets.Production, nil
	default:
		if config, exists := manifest.Secrets.Environments[environment]; exists {
			return &config, nil
		}
		return nil, fmt.Errorf("environment %s not found", environment)
	}
}

// Private validation methods

func (m *Manager) validateTool(name string, tool ToolConfig) error {
	// No tool type validation needed in dynamic loading approach
	// Tools are automatically discovered and loaded from src/tools/ directory
	return nil
}

func (m *Manager) validateResource(name string, resource ResourceConfig) error {
	// Add resource validation logic as needed
	return nil
}

func (m *Manager) validateSecrets(secrets SecretsConfig) error {
	// Validate each secret provider configuration
	configs := []SecretProviderConfig{
		secrets.Local,
		secrets.Staging,
		secrets.Production,
	}

	for _, config := range configs {
		if config.Provider != "" && !isValidSecretProvider(config.Provider) {
			return fmt.Errorf("invalid secret provider: %s", config.Provider)
		}
	}

	// Validate custom environments
	for env, config := range secrets.Environments {
		if config.Provider != "" && !isValidSecretProvider(config.Provider) {
			return fmt.Errorf("invalid secret provider for environment %s: %s", env, config.Provider)
		}
	}

	return nil
}

// Helper functions

func isValidFramework(framework string) bool {
	validFrameworks := []string{
		FrameworkFastMCPPython,
		FrameworkFastMCPTypeScript,
		FrameworkEasyMCPTypeScript,
		FrameworkOfficialPython,
		FrameworkOfficialTypeScript,
	}

	for _, valid := range validFrameworks {
		if framework == valid {
			return true
		}
	}
	return false
}

// isValidToolType is no longer needed in the dynamic loading approach
// but kept for backwards compatibility with other frameworks
func isValidToolType(toolType string) bool {
	// For dynamic loading (FastMCP Python), tool types are not used
	// Return true for any value to avoid validation errors
	return true
}

func isValidSecretProvider(provider string) bool {
	validProviders := []string{
		SecretProviderEnv,
		SecretProviderKubernetes,
	}

	for _, valid := range validProviders {
		if provider == valid {
			return true
		}
	}
	return false
}
