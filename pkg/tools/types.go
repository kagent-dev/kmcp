package tools

import "fmt"

// ToolInfo represents information about a discovered or generated tool
type ToolInfo struct {
	Name        string                 `json:"name"`
	FilePath    string                 `json:"file_path"`
	ClassName   string                 `json:"class_name"`
	Description string                 `json:"description"`
	Methods     []MethodInfo           `json:"methods"`
	Imports     []string               `json:"imports"`
	Config      map[string]interface{} `json:"config"`
}

// MethodInfo represents information about a tool method
type MethodInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  []ParameterInfo        `json:"parameters"`
	ReturnType  string                 `json:"return_type"`
	IsAsync     bool                   `json:"is_async"`
	Config      map[string]interface{} `json:"config"`
}

// ParameterInfo represents information about a method parameter
type ParameterInfo struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default"`
}

// ToolTemplate represents a template for generating tools
type ToolTemplate struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Description  string            `json:"description"`
	Content      string            `json:"content"`
	Variables    map[string]string `json:"variables"`
	Dependencies []string          `json:"dependencies"`
}

// Simplified tool type definitions
const (
	ToolTypeBasic    = "basic"
	ToolTypeHTTP     = "http"
	ToolTypeData     = "data"
	ToolTypeWorkflow = "workflow"
)

// ToolTypeConfig represents configuration schema for a tool type
type ToolTypeConfig struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Dependencies []string               `json:"dependencies"`
	ConfigSchema map[string]interface{} `json:"config_schema"`
	Examples     map[string]interface{} `json:"examples"`
}

// GetToolTypeConfig returns the configuration schema for a tool type
func GetToolTypeConfig(toolType string) (*ToolTypeConfig, error) {
	configs := GetAllToolTypeConfigs()
	config, exists := configs[toolType]
	if !exists {
		return nil, fmt.Errorf("unknown tool type: %s", toolType)
	}
	return &config, nil
}

// GetAllToolTypeConfigs returns all available tool type configurations
func GetAllToolTypeConfigs() map[string]ToolTypeConfig {
	return map[string]ToolTypeConfig{
		ToolTypeBasic: {
			Name:         "Basic Tool",
			Description:  "Minimal tool structure with basic functionality",
			Dependencies: []string{},
			ConfigSchema: map[string]interface{}{
				"enabled": map[string]interface{}{
					"type":        "boolean",
					"description": "Enable/disable the tool",
					"default":     true,
				},
			},
			Examples: map[string]interface{}{
				"enabled": true,
			},
		},
		ToolTypeHTTP: {
			Name:         "HTTP Tool",
			Description:  "Tool with HTTP client capabilities",
			Dependencies: []string{"httpx"},
			ConfigSchema: map[string]interface{}{
				"enabled": map[string]interface{}{
					"type":        "boolean",
					"description": "Enable/disable the tool",
					"default":     true,
				},
				"base_url": map[string]interface{}{
					"type":        "string",
					"description": "Base URL for HTTP requests",
					"default":     "",
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"description": "Request timeout in seconds",
					"default":     30,
				},
			},
			Examples: map[string]interface{}{
				"base_url": "https://api.example.com",
				"timeout":  30,
				"enabled":  true,
			},
		},
		ToolTypeData: {
			Name:         "Data Tool",
			Description:  "Tool for data processing and manipulation",
			Dependencies: []string{},
			ConfigSchema: map[string]interface{}{
				"enabled": map[string]interface{}{
					"type":        "boolean",
					"description": "Enable/disable the tool",
					"default":     true,
				},
			},
			Examples: map[string]interface{}{
				"enabled": true,
			},
		},
		ToolTypeWorkflow: {
			Name:         "Workflow Tool",
			Description:  "Tool for multi-step operations and workflows",
			Dependencies: []string{},
			ConfigSchema: map[string]interface{}{
				"enabled": map[string]interface{}{
					"type":        "boolean",
					"description": "Enable/disable the tool",
					"default":     true,
				},
				"max_steps": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of workflow steps",
					"default":     10,
				},
			},
			Examples: map[string]interface{}{
				"max_steps": 10,
				"enabled":   true,
			},
		},
	}
}

// GetValidToolTypes returns all valid tool type names
func GetValidToolTypes() []string {
	return []string{
		ToolTypeBasic,
		ToolTypeHTTP,
		ToolTypeData,
		ToolTypeWorkflow,
	}
}

// IsValidToolType checks if a tool type is valid
func IsValidToolType(toolType string) bool {
	validTypes := GetValidToolTypes()
	for _, valid := range validTypes {
		if toolType == valid {
			return true
		}
	}
	return false
}
