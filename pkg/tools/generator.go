package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Generator handles generating new Python tool files from templates
type Generator struct {
	templates map[string]*ToolTemplate
}

// NewGenerator creates a new generator instance
func NewGenerator() *Generator {
	g := &Generator{
		templates: make(map[string]*ToolTemplate),
	}

	// Initialize built-in templates
	g.initBuiltinTemplates()

	return g
}

// GenerateToolFile generates a new Python tool file from a template
func (g *Generator) GenerateToolFile(filePath, toolName, toolType string, config map[string]interface{}) error {
	// Get the template for the tool type
	toolTemplate, exists := g.templates[toolType]
	if !exists {
		return fmt.Errorf("unknown tool type: %s", toolType)
	}

	// Prepare template data
	data := map[string]interface{}{
		"ToolName":      toolName,
		"ToolNameTitle": strings.Title(toolName),
		"ToolNameUpper": strings.ToUpper(toolName),
		"ToolNameLower": strings.ToLower(toolName),
		"ClassName":     strings.Title(toolName) + "Tool",
		"Config":        config,
	}

	// Add config values to template data
	for key, value := range config {
		data[key] = value
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Parse and execute the template
	tmpl, err := template.New(toolType).Parse(toolTemplate.Content)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create the output file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Execute the template
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// initBuiltinTemplates initializes the built-in tool templates
func (g *Generator) initBuiltinTemplates() {
	// Basic tool template - minimal structure
	g.templates["basic"] = &ToolTemplate{
		Name:         "basic",
		Type:         "basic",
		Description:  "Minimal tool structure with basic functionality",
		Content:      g.getBasicTemplate(),
		Dependencies: []string{},
	}

	// HTTP tool template - basic HTTP client
	g.templates["http"] = &ToolTemplate{
		Name:         "http",
		Type:         "http",
		Description:  "Tool with HTTP client capabilities",
		Content:      g.getHTTPTemplate(),
		Dependencies: []string{"httpx"},
	}

	// Data tool template - data processing focus
	g.templates["data"] = &ToolTemplate{
		Name:         "data",
		Type:         "data",
		Description:  "Tool for data processing and manipulation",
		Content:      g.getDataTemplate(),
		Dependencies: []string{},
	}

	// Workflow tool template - multi-step operations
	g.templates["workflow"] = &ToolTemplate{
		Name:         "workflow",
		Type:         "workflow",
		Description:  "Tool for multi-step operations and workflows",
		Content:      g.getWorkflowTemplate(),
		Dependencies: []string{},
	}
}

// getBasicTemplate returns the minimal basic tool template
func (g *Generator) getBasicTemplate() string {
	return `"""{{.ToolNameTitle}} tool for MCP server."""

from typing import Dict, Any


class {{.ClassName}}:
    """{{.ToolNameTitle}} tool implementation.{{if .description}}
    
    {{.description}}{{end}}"""
    
    def __init__(self, config: Dict[str, Any] = None):
        """Initialize the {{.ToolName}} tool."""
        self.config = config or {}
    
    async def process(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """Process data and return results.
        
        Args:
            data: Input data to process
            
        Returns:
            Dict containing the processed results
        """
        # TODO: Implement your tool logic here
        return {
            "tool": "{{.ToolName}}",
            "input": data,
            "result": "Processed successfully"
        }
    
    async def get_status(self) -> Dict[str, Any]:
        """Get tool status."""
        return {
            "tool": "{{.ToolName}}",
            "status": "ready"
        }
`
}

// getHTTPTemplate returns the HTTP tool template
func (g *Generator) getHTTPTemplate() string {
	return `"""{{.ToolNameTitle}} HTTP tool for MCP server."""

from typing import Dict, Any, Optional
import httpx


class {{.ClassName}}:
    """{{.ToolNameTitle}} HTTP tool implementation.{{if .description}}
    
    {{.description}}{{end}}"""
    
    def __init__(self, config: Dict[str, Any] = None):
        """Initialize the {{.ToolName}} HTTP tool."""
        self.config = config or {}
        self.base_url = self.config.get("base_url", "{{.base_url}}")
        self.timeout = self.config.get("timeout", 30)
    
    async def make_request(self, method: str, endpoint: str, **kwargs) -> Dict[str, Any]:
        """Make an HTTP request.
        
        Args:
            method: HTTP method (GET, POST, etc.)
            endpoint: API endpoint
            **kwargs: Additional request parameters
            
        Returns:
            Dict containing the response data
        """
        url = f"{self.base_url.rstrip('/')}/{endpoint.lstrip('/')}" if self.base_url else endpoint
        
        try:
            async with httpx.AsyncClient(timeout=self.timeout) as client:
                response = await client.request(method, url, **kwargs)
                
                # Try to parse JSON, fall back to text
                try:
                    data = response.json()
                except:
                    data = response.text
                
                return {
                    "status_code": response.status_code,
                    "data": data,
                    "success": response.is_success
                }
        except Exception as e:
            return {
                "error": str(e),
                "success": False
            }
    
    async def get_status(self) -> Dict[str, Any]:
        """Get tool status."""
        return {
            "tool": "{{.ToolName}}",
            "base_url": self.base_url,
            "timeout": self.timeout,
            "status": "ready"
        }
`
}

// getDataTemplate returns the data processing tool template
func (g *Generator) getDataTemplate() string {
	return `"""{{.ToolNameTitle}} data processing tool for MCP server."""

from typing import Dict, Any, List, Union


class {{.ClassName}}:
    """{{.ToolNameTitle}} data processing tool implementation.{{if .description}}
    
    {{.description}}{{end}}"""
    
    def __init__(self, config: Dict[str, Any] = None):
        """Initialize the {{.ToolName}} data tool."""
        self.config = config or {}
    
    async def process_data(self, data: Union[Dict, List, str]) -> Dict[str, Any]:
        """Process input data.
        
        Args:
            data: Input data to process
            
        Returns:
            Dict containing the processed results
        """
        # TODO: Implement your data processing logic here
        return {
            "tool": "{{.ToolName}}",
            "input_type": type(data).__name__,
            "result": "Data processed successfully"
        }
    
    async def validate_data(self, data: Any) -> Dict[str, Any]:
        """Validate input data.
        
        Args:
            data: Data to validate
            
        Returns:
            Dict containing validation results
        """
        # TODO: Implement your validation logic here
        return {
            "valid": True,
            "errors": []
        }
    
    async def get_status(self) -> Dict[str, Any]:
        """Get tool status."""
        return {
            "tool": "{{.ToolName}}",
            "status": "ready"
        }
`
}

// getWorkflowTemplate returns the workflow tool template
func (g *Generator) getWorkflowTemplate() string {
	return `"""{{.ToolNameTitle}} workflow tool for MCP server."""

from typing import Dict, Any, List


class {{.ClassName}}:
    """{{.ToolNameTitle}} workflow tool implementation.{{if .description}}
    
    {{.description}}{{end}}"""
    
    def __init__(self, config: Dict[str, Any] = None):
        """Initialize the {{.ToolName}} workflow tool."""
        self.config = config or {}
        self.max_steps = self.config.get("max_steps", 10)
    
    async def execute_workflow(self, steps: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Execute a workflow with multiple steps.
        
        Args:
            steps: List of workflow steps to execute
            
        Returns:
            Dict containing the workflow results
        """
        if len(steps) > self.max_steps:
            return {
                "error": f"Too many steps (max {self.max_steps})"
            }
        
        results = []
        context = {}
        
        for i, step in enumerate(steps):
            # TODO: Implement your step execution logic here
            step_result = await self.execute_step(step, context)
            results.append(step_result)
            
            # Update context with step results
            if step_result.get("context"):
                context.update(step_result["context"])
        
        return {
            "tool": "{{.ToolName}}",
            "steps_executed": len(results),
            "results": results,
            "context": context
        }
    
    async def execute_step(self, step: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a single workflow step.
        
        Args:
            step: Step configuration
            context: Current workflow context
            
        Returns:
            Dict containing step results
        """
        # TODO: Implement your step execution logic here
        return {
            "step_type": step.get("type", "unknown"),
            "status": "success",
            "context": {}
        }
    
    async def get_status(self) -> Dict[str, Any]:
        """Get tool status."""
        return {
            "tool": "{{.ToolName}}",
            "max_steps": self.max_steps,
            "status": "ready"
        }
`
}

// GetTemplate returns a template by name
func (g *Generator) GetTemplate(name string) (*ToolTemplate, bool) {
	template, exists := g.templates[name]
	return template, exists
}

// ListTemplates returns all available templates
func (g *Generator) ListTemplates() []string {
	names := make([]string, 0, len(g.templates))
	for name := range g.templates {
		names = append(names, name)
	}
	return names
}
