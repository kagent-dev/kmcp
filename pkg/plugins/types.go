package plugins

import (
	"context"
)

// Tool represents an MCP tool plugin
type Tool interface {
	// Name returns the unique name of this tool
	Name() string

	// Description returns a human-readable description of the tool
	Description() string

	// Methods returns the list of available methods for this tool
	Methods() []MethodInfo

	// Call executes a method with the given parameters
	Call(ctx context.Context, method string, params map[string]interface{}) (*CallResult, error)

	// Dependencies returns the list of dependencies this tool requires
	Dependencies() []string

	// Config returns the configuration schema for this tool
	Config() ToolConfig

	// Initialize sets up the tool with the given configuration
	Initialize(config map[string]interface{}) error

	// IsEnabled returns whether this tool is currently enabled
	IsEnabled() bool

	// SetEnabled sets the enabled state of this tool
	SetEnabled(enabled bool)
}

// Resource represents an MCP resource plugin
type Resource interface {
	// Name returns the unique name of this resource
	Name() string

	// Description returns a human-readable description of the resource
	Description() string

	// URITemplate returns the URI template for this resource
	URITemplate() string

	// MimeTypes returns the supported MIME types for this resource
	MimeTypes() []string

	// Read reads the resource content for the given URI
	Read(ctx context.Context, uri string) (*ResourceContent, error)

	// List lists available resources (if supported)
	List(ctx context.Context, cursor string) (*ResourceList, error)

	// Config returns the configuration schema for this resource
	Config() ResourceConfig

	// Initialize sets up the resource with the given configuration
	Initialize(config map[string]interface{}) error

	// IsEnabled returns whether this resource is currently enabled
	IsEnabled() bool

	// SetEnabled sets the enabled state of this resource
	SetEnabled(enabled bool)
}

// MethodInfo describes a tool method
type MethodInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Required    []string               `json:"required,omitempty"`
}

// CallResult represents the result of a tool method call
type CallResult struct {
	Content interface{} `json:"content"`
	IsText  bool        `json:"isText"`
	Error   string      `json:"error,omitempty"`
}

// ResourceContent represents the content of a resource
type ResourceContent struct {
	URI      string                 `json:"uri"`
	MimeType string                 `json:"mimeType"`
	Text     string                 `json:"text,omitempty"`
	Blob     []byte                 `json:"blob,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceList represents a list of resources
type ResourceList struct {
	Resources []ResourceInfo `json:"resources"`
	NextToken string         `json:"nextToken,omitempty"`
}

// ResourceInfo describes a resource
type ResourceInfo struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ToolConfig represents tool configuration
type ToolConfig struct {
	Type         string                 `json:"type"`
	Schema       map[string]interface{} `json:"schema,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Enabled      bool                   `json:"enabled"`
}

// ResourceConfig represents resource configuration
type ResourceConfig struct {
	Type    string                 `json:"type"`
	Schema  map[string]interface{} `json:"schema,omitempty"`
	Enabled bool                   `json:"enabled"`
}

// PluginFactory creates plugin instances
type PluginFactory interface {
	// CreateTool creates a new tool instance
	CreateTool(toolType string, config map[string]interface{}) (Tool, error)

	// CreateResource creates a new resource instance
	CreateResource(resourceType string, config map[string]interface{}) (Resource, error)

	// ListToolTypes returns available tool types
	ListToolTypes() []string

	// ListResourceTypes returns available resource types
	ListResourceTypes() []string
}

// Context provides shared services to plugins
type Context struct {
	// SecretManager provides access to secrets
	SecretManager SecretManager

	// Logger provides logging capabilities
	Logger Logger

	// Config provides access to plugin configuration
	Config map[string]interface{}

	// ProjectRoot is the root directory of the project
	ProjectRoot string
}

// SecretManager interface for accessing secrets
type SecretManager interface {
	Get(key string) (string, error)
	GetAll() (map[string]string, error)
	Exists(key string) bool
	ListKeys() ([]string, error)
	SanitizeForMCP(data interface{}) interface{}
}

// Logger interface for plugin logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// Plugin metadata
type PluginMetadata struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Homepage    string                 `json:"homepage,omitempty"`
	Repository  string                 `json:"repository,omitempty"`
	License     string                 `json:"license,omitempty"`
	Keywords    []string               `json:"keywords,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// PluginRegistry manages plugin registration and discovery
type PluginRegistry interface {
	// RegisterTool registers a tool type
	RegisterTool(toolType string, factory func() Tool) error

	// RegisterResource registers a resource type
	RegisterResource(resourceType string, factory func() Resource) error

	// GetTool creates a tool instance by type
	GetTool(toolType string) (Tool, error)

	// GetResource creates a resource instance by type
	GetResource(resourceType string) (Resource, error)

	// ListTools returns all registered tool types
	ListTools() []string

	// ListResources returns all registered resource types
	ListResources() []string

	// GetMetadata returns plugin metadata
	GetMetadata(pluginName string) (*PluginMetadata, error)
}
