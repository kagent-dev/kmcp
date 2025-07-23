package plugins

import (
	"fmt"
	"sync"
)

// DefaultRegistry provides the default plugin registry implementation
type DefaultRegistry struct {
	toolFactories     map[string]func() Tool
	resourceFactories map[string]func() Resource
	metadata          map[string]*PluginMetadata
	mu                sync.RWMutex
}

// NewRegistry creates a new plugin registry
func NewRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		toolFactories:     make(map[string]func() Tool),
		resourceFactories: make(map[string]func() Resource),
		metadata:          make(map[string]*PluginMetadata),
	}
}

// RegisterTool registers a tool type with a factory function
func (r *DefaultRegistry) RegisterTool(toolType string, factory func() Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.toolFactories[toolType]; exists {
		return fmt.Errorf("tool type %s is already registered", toolType)
	}

	r.toolFactories[toolType] = factory
	return nil
}

// RegisterResource registers a resource type with a factory function
func (r *DefaultRegistry) RegisterResource(resourceType string, factory func() Resource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.resourceFactories[resourceType]; exists {
		return fmt.Errorf("resource type %s is already registered", resourceType)
	}

	r.resourceFactories[resourceType] = factory
	return nil
}

// GetTool creates a tool instance by type
func (r *DefaultRegistry) GetTool(toolType string) (Tool, error) {
	r.mu.RLock()
	factory, exists := r.toolFactories[toolType]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool type %s not found", toolType)
	}

	return factory(), nil
}

// GetResource creates a resource instance by type
func (r *DefaultRegistry) GetResource(resourceType string) (Resource, error) {
	r.mu.RLock()
	factory, exists := r.resourceFactories[resourceType]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("resource type %s not found", resourceType)
	}

	return factory(), nil
}

// ListTools returns all registered tool types
func (r *DefaultRegistry) ListTools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]string, len(r.toolFactories))
	for toolType := range r.toolFactories {
		tools = append(tools, toolType)
	}

	return tools
}

// ListResources returns all registered resource types
func (r *DefaultRegistry) ListResources() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resources := make([]string, len(r.resourceFactories))
	for resourceType := range r.resourceFactories {
		resources = append(resources, resourceType)
	}

	return resources
}

// RegisterMetadata registers metadata for a plugin
func (r *DefaultRegistry) RegisterMetadata(pluginName string, metadata *PluginMetadata) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metadata[pluginName] = metadata
}

// GetMetadata returns plugin metadata
func (r *DefaultRegistry) GetMetadata(pluginName string) (*PluginMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[pluginName]
	if !exists {
		return nil, fmt.Errorf("metadata for plugin %s not found", pluginName)
	}

	return metadata, nil
}

// Manager manages plugin lifecycle and provides runtime services
type Manager struct {
	registry  PluginRegistry
	context   *Context
	tools     map[string]Tool
	resources map[string]Resource
	mu        sync.RWMutex
}

// NewManager creates a new plugin manager
func NewManager(registry PluginRegistry, context *Context) *Manager {
	return &Manager{
		registry:  registry,
		context:   context,
		tools:     make(map[string]Tool),
		resources: make(map[string]Resource),
	}
}

// LoadTool loads and initializes a tool
func (m *Manager) LoadTool(name, toolType string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if tool is already loaded
	if _, exists := m.tools[name]; exists {
		return fmt.Errorf("tool %s is already loaded", name)
	}

	// Create tool instance
	tool, err := m.registry.GetTool(toolType)
	if err != nil {
		return fmt.Errorf("failed to create tool %s: %w", name, err)
	}

	// Initialize tool with configuration
	if err := tool.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize tool %s: %w", name, err)
	}

	m.tools[name] = tool
	return nil
}

// LoadResource loads and initializes a resource
func (m *Manager) LoadResource(name, resourceType string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if resource is already loaded
	if _, exists := m.resources[name]; exists {
		return fmt.Errorf("resource %s is already loaded", name)
	}

	// Create resource instance
	resource, err := m.registry.GetResource(resourceType)
	if err != nil {
		return fmt.Errorf("failed to create resource %s: %w", name, err)
	}

	// Initialize resource with configuration
	if err := resource.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize resource %s: %w", name, err)
	}

	m.resources[name] = resource
	return nil
}

// GetTool returns a loaded tool by name
func (m *Manager) GetTool(name string) (Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tool, exists := m.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return tool, nil
}

// GetResource returns a loaded resource by name
func (m *Manager) GetResource(name string) (Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	resource, exists := m.resources[name]
	if !exists {
		return nil, fmt.Errorf("resource %s not found", name)
	}

	return resource, nil
}

// ListLoadedTools returns all loaded tool names
func (m *Manager) ListLoadedTools() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]string, len(m.tools))
	for name := range m.tools {
		if m.tools[name].IsEnabled() {
			tools = append(tools, name)
		}
	}

	return tools
}

// ListLoadedResources returns all loaded resource names
func (m *Manager) ListLoadedResources() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	resources := make([]string, len(m.resources))
	for name := range m.resources {
		if m.resources[name].IsEnabled() {
			resources = append(resources, name)
		}
	}

	return resources
}

// UnloadTool removes a tool from the manager
func (m *Manager) UnloadTool(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}

	delete(m.tools, name)
	return nil
}

// UnloadResource removes a resource from the manager
func (m *Manager) UnloadResource(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.resources[name]; !exists {
		return fmt.Errorf("resource %s not found", name)
	}

	delete(m.resources, name)
	return nil
}

// EnableTool enables a tool
func (m *Manager) EnableTool(name string) error {
	tool, err := m.GetTool(name)
	if err != nil {
		return err
	}

	tool.SetEnabled(true)
	return nil
}

// DisableTool disables a tool
func (m *Manager) DisableTool(name string) error {
	tool, err := m.GetTool(name)
	if err != nil {
		return err
	}

	tool.SetEnabled(false)
	return nil
}

// EnableResource enables a resource
func (m *Manager) EnableResource(name string) error {
	resource, err := m.GetResource(name)
	if err != nil {
		return err
	}

	resource.SetEnabled(true)
	return nil
}

// DisableResource disables a resource
func (m *Manager) DisableResource(name string) error {
	resource, err := m.GetResource(name)
	if err != nil {
		return err
	}

	resource.SetEnabled(false)
	return nil
}

// GetContext returns the plugin context
func (m *Manager) GetContext() *Context {
	return m.context
}

// Global registry instance
var globalRegistry PluginRegistry = NewRegistry()

// GetGlobalRegistry returns the global plugin registry
func GetGlobalRegistry() PluginRegistry {
	return globalRegistry
}

// RegisterGlobalTool registers a tool in the global registry
func RegisterGlobalTool(toolType string, factory func() Tool) error {
	return globalRegistry.RegisterTool(toolType, factory)
}

// RegisterGlobalResource registers a resource in the global registry
func RegisterGlobalResource(resourceType string, factory func() Resource) error {
	return globalRegistry.RegisterResource(resourceType, factory)
}

// RegisterGlobalMetadata registers metadata in the global registry
func RegisterGlobalMetadata(pluginName string, metadata *PluginMetadata) {
	if dr, ok := globalRegistry.(*DefaultRegistry); ok {
		dr.RegisterMetadata(pluginName, metadata)
	}
}
