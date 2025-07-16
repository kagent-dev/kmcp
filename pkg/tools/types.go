package tools

// ToolInfo represents information about a discovered or generated tool
type ToolInfo struct {
	Name         string                 `json:"name"`
	FilePath     string                 `json:"file_path"`
	FunctionName string                 `json:"function_name"`
	Description  string                 `json:"description"`
	Parameters   []ParameterInfo        `json:"parameters"`
	ReturnType   string                 `json:"return_type"`
	IsAsync      bool                   `json:"is_async"`
	Config       map[string]interface{} `json:"config"`
}

// ParameterInfo represents information about a function parameter
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
	Description  string            `json:"description"`
	Content      string            `json:"content"`
	Variables    map[string]string `json:"variables"`
	Dependencies []string          `json:"dependencies"`
}
