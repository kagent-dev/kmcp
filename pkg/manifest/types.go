package manifest

import (
	"time"
)

// ProjectManifest represents the complete kmcp.yaml configuration
type ProjectManifest struct {
	// Project metadata
	Name        string `yaml:"name" json:"name"`
	Framework   string `yaml:"framework" json:"framework"`
	Version     string `yaml:"version" json:"version"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Author      string `yaml:"author,omitempty" json:"author,omitempty"`
	Email       string `yaml:"email,omitempty" json:"email,omitempty"`

	// Project configuration
	Tools     map[string]ToolConfig     `yaml:"tools,omitempty" json:"tools,omitempty"`
	Resources map[string]ResourceConfig `yaml:"resources,omitempty" json:"resources,omitempty"`
	Secrets   SecretsConfig             `yaml:"secrets,omitempty" json:"secrets,omitempty"`

	// Dependency management
	Dependencies DependencyConfig `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`

	// Build configuration
	Build BuildConfig `yaml:"build,omitempty" json:"build,omitempty"`

	// Metadata
	CreatedAt time.Time `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt time.Time `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// ToolConfig represents configuration for an MCP tool
type ToolConfig struct {
	Name        string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Handler     string                 `yaml:"handler,omitempty" json:"handler,omitempty"`
	Enabled     bool                   `yaml:"enabled" json:"enabled"`
	Type        string                 `yaml:"type,omitempty" json:"type,omitempty"`
	Template    string                 `yaml:"template,omitempty" json:"template,omitempty"`
	Config      map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`

	// Dependencies for this tool
	Dependencies []string `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
}

// ResourceConfig represents configuration for an MCP resource
type ResourceConfig struct {
	Enabled     bool                   `yaml:"enabled" json:"enabled"`
	Type        string                 `yaml:"type,omitempty" json:"type,omitempty"`
	Config      map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
}

// SecretsConfig represents secret management configuration
type SecretsConfig struct {
	// Environment-specific secret configurations
	Local      SecretProviderConfig `yaml:"local,omitempty" json:"local,omitempty"`
	Staging    SecretProviderConfig `yaml:"staging,omitempty" json:"staging,omitempty"`
	Production SecretProviderConfig `yaml:"production,omitempty" json:"production,omitempty"`

	// Custom environments
	Environments map[string]SecretProviderConfig `yaml:"environments,omitempty" json:"environments,omitempty"`
}

// SecretProviderConfig represents configuration for a secret provider
type SecretProviderConfig struct {
	Provider string                 `yaml:"provider" json:"provider"` // env, kubernetes
	Config   map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`

	// For environment provider
	Source string `yaml:"source,omitempty" json:"source,omitempty"` // .env.local

	// For kubernetes provider
	SecretName string `yaml:"secret_name,omitempty" json:"secret_name,omitempty"`
	Namespace  string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
}

// DependencyConfig represents dependency management configuration
type DependencyConfig struct {
	AutoManage bool     `yaml:"auto_manage" json:"auto_manage"`
	Runtime    []string `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Dev        []string `yaml:"dev,omitempty" json:"dev,omitempty"`
	Extra      []string `yaml:"extra,omitempty" json:"extra,omitempty"`
}

// BuildConfig represents build configuration
type BuildConfig struct {
	Output   string       `yaml:"output,omitempty" json:"output,omitempty"`
	Docker   DockerConfig `yaml:"docker,omitempty" json:"docker,omitempty"`
	Target   string       `yaml:"target,omitempty" json:"target,omitempty"`
	Platform string       `yaml:"platform,omitempty" json:"platform,omitempty"`
}

// DockerConfig represents Docker build configuration
type DockerConfig struct {
	Image       string            `yaml:"image,omitempty" json:"image,omitempty"`
	Dockerfile  string            `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`
	Platform    []string          `yaml:"platform,omitempty" json:"platform,omitempty"`
	BaseImage   string            `yaml:"base_image,omitempty" json:"base_image,omitempty"`
	Port        int               `yaml:"port,omitempty" json:"port,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	HealthCheck string            `yaml:"health_check,omitempty" json:"health_check,omitempty"`
}

// Supported frameworks
const (
	FrameworkFastMCPPython      = "fastmcp-python"
	FrameworkFastMCPTypeScript  = "fastmcp-ts"
	FrameworkEasyMCPTypeScript  = "easymcp-ts"
	FrameworkOfficialPython     = "official-python"
	FrameworkOfficialTypeScript = "official-ts"
)

// Supported secret providers
const (
	SecretProviderEnv        = "env"
	SecretProviderKubernetes = "kubernetes"
)

// Supported tool types
const (
	ToolTypeBasic      = "basic"
	ToolTypeDatabase   = "database"
	ToolTypeFilesystem = "filesystem"
	ToolTypeAPIClient  = "api-client"
	ToolTypeMultiTool  = "multi-tool"
)
