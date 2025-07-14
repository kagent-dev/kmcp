package secrets

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kagent.dev/kmcp/pkg/manifest"
	"kagent.dev/kmcp/pkg/security/sanitizer"
)

// Manager handles secure secret access across different providers
type Manager struct {
	environment string
	config      *manifest.SecretProviderConfig
	k8sClient   kubernetes.Interface
	envVars     map[string]string
}

// NewManager creates a new secret manager for the specified environment
func NewManager(environment string, config *manifest.SecretProviderConfig) (*Manager, error) {
	manager := &Manager{
		environment: environment,
		config:      config,
		envVars:     make(map[string]string),
	}

	// Initialize the appropriate provider
	switch config.Provider {
	case manifest.SecretProviderKubernetes:
		if err := manager.initKubernetes(); err != nil {
			return nil, fmt.Errorf("failed to initialize Kubernetes client: %w", err)
		}
	case manifest.SecretProviderEnv:
		if err := manager.initEnvironment(); err != nil {
			return nil, fmt.Errorf("failed to initialize environment provider: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported secret provider: %s", config.Provider)
	}

	return manager, nil
}

// Get retrieves a secret value by key
func (m *Manager) Get(key string) (string, error) {
	switch m.config.Provider {
	case manifest.SecretProviderKubernetes:
		return m.getFromKubernetes(key)
	case manifest.SecretProviderEnv:
		return m.getFromEnvironment(key)
	default:
		return "", fmt.Errorf("unsupported secret provider: %s", m.config.Provider)
	}
}

// GetAll retrieves all available secrets
func (m *Manager) GetAll() (map[string]string, error) {
	switch m.config.Provider {
	case manifest.SecretProviderKubernetes:
		return m.getAllFromKubernetes()
	case manifest.SecretProviderEnv:
		return m.getAllFromEnvironment()
	default:
		return nil, fmt.Errorf("unsupported secret provider: %s", m.config.Provider)
	}
}

// Set stores a secret value (only supported for environment provider)
func (m *Manager) Set(key, value string) error {
	switch m.config.Provider {
	case manifest.SecretProviderEnv:
		return m.setInEnvironment(key, value)
	case manifest.SecretProviderKubernetes:
		return fmt.Errorf("setting secrets in Kubernetes provider not supported; use kubectl or Kubernetes API directly")
	default:
		return fmt.Errorf("unsupported secret provider: %s", m.config.Provider)
	}
}

// Exists checks if a secret exists
func (m *Manager) Exists(key string) bool {
	_, err := m.Get(key)
	return err == nil
}

// ListKeys returns all available secret keys
func (m *Manager) ListKeys() ([]string, error) {
	secrets, err := m.GetAll()
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(secrets))
	for key := range secrets {
		keys = append(keys, key)
	}

	return keys, nil
}

// Kubernetes provider methods

func (m *Manager) initKubernetes() error {
	config, err := m.getKubernetesConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	m.k8sClient = clientset
	return nil
}

func (m *Manager) getKubernetesConfig() (*rest.Config, error) {
	// Try in-cluster config first
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}

	// Fall back to kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfig = fmt.Sprintf("%s/.kube/config", homeDir)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	return config, nil
}

func (m *Manager) getFromKubernetes(key string) (string, error) {
	secret, err := m.k8sClient.CoreV1().Secrets(m.config.Namespace).Get(
		context.TODO(),
		m.config.SecretName,
		metav1.GetOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", m.config.Namespace, m.config.SecretName, err)
	}

	value, exists := secret.Data[key]
	if !exists {
		return "", fmt.Errorf("key %s not found in secret %s/%s", key, m.config.Namespace, m.config.SecretName)
	}

	return string(value), nil
}

func (m *Manager) getAllFromKubernetes() (map[string]string, error) {
	secret, err := m.k8sClient.CoreV1().Secrets(m.config.Namespace).Get(
		context.TODO(),
		m.config.SecretName,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", m.config.Namespace, m.config.SecretName, err)
	}

	result := make(map[string]string)
	for key, value := range secret.Data {
		result[key] = string(value)
	}

	return result, nil
}

// Environment provider methods

func (m *Manager) initEnvironment() error {
	// Load from specified source file if provided
	if m.config.Source != "" {
		if err := godotenv.Load(m.config.Source); err != nil {
			// Don't fail if file doesn't exist for .env files
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to load environment file %s: %w", m.config.Source, err)
			}
		}
	}

	// Load environment variables
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			m.envVars[pair[0]] = pair[1]
		}
	}

	return nil
}

func (m *Manager) getFromEnvironment(key string) (string, error) {
	value, exists := m.envVars[key]
	if !exists {
		// Try to get from current environment as fallback
		value = os.Getenv(key)
		if value == "" {
			return "", fmt.Errorf("environment variable %s not found", key)
		}
	}

	return value, nil
}

func (m *Manager) getAllFromEnvironment() (map[string]string, error) {
	// Return a copy to prevent external modification
	result := make(map[string]string)
	for key, value := range m.envVars {
		result[key] = value
	}

	return result, nil
}

func (m *Manager) setInEnvironment(key, value string) error {
	m.envVars[key] = value

	// Also set in the process environment
	return os.Setenv(key, value)
}

// Utility methods

// CreateKubernetesSecret creates a Kubernetes secret manifest
func (m *Manager) CreateKubernetesSecret(secrets map[string]string) (*corev1.Secret, error) {
	if m.config.Provider != manifest.SecretProviderKubernetes {
		return nil, fmt.Errorf("can only create Kubernetes secrets for Kubernetes provider")
	}

	data := make(map[string][]byte)
	for key, value := range secrets {
		data[key] = []byte(value)
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.config.SecretName,
			Namespace: m.config.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}

	return secret, nil
}

// SanitizeForMCP removes sensitive values from data before sending to MCP
func (m *Manager) SanitizeForMCP(data interface{}) interface{} {
	s := sanitizer.NewSanitizer()
	return s.Sanitize(data)
}
