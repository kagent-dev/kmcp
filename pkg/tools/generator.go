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
}

// NewGenerator creates a new generator instance
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateToolFile generates a new Python tool file from the unified template
func (g *Generator) GenerateToolFile(filePath, toolName string, config map[string]interface{}) error {
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
	tmpl, err := template.New("tool").Parse(g.getUnifiedTemplate())
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

// getUnifiedTemplate returns the unified tool template with commented examples
func (g *Generator) getUnifiedTemplate() string {
	return `"""{{.ToolNameTitle}} tool for MCP server.{{if .description}}

{{.description}}{{end}}

This tool is automatically loaded by the FastMCP dynamic loading system.
The function name must match the filename for auto-discovery.
"""

from core.server import mcp
from core.utils import get_tool_config, get_env_var

# Import additional dependencies as needed
# import httpx          # For HTTP requests
# import asyncpg        # For PostgreSQL database
# import aiofiles       # For async file operations
# import json           # For JSON processing
# import yaml           # For YAML processing


@mcp.tool()
def {{.ToolName}}(message: str) -> str:
    """{{.ToolNameTitle}} tool implementation.
    
    This is a template function. Replace this implementation with your tool logic.
    
    Args:
        message: Input message (replace with your actual parameters)
        
    Returns:
        str: Result of the tool operation (replace with your actual return type)
    """
    # Get tool-specific configuration from kmcp.yaml
    config = get_tool_config("{{.ToolName}}")
    
    # TODO: Replace this basic implementation with your tool logic
    
    # Example: Basic text processing
    prefix = config.get("prefix", "")
    return f"{prefix}{message}"
    
    # Example: HTTP API call
    # api_key = get_env_var(config.get("api_key_env", "API_KEY"))
    # base_url = config.get("base_url", "https://api.example.com")
    # timeout = config.get("timeout", 30)
    # 
    # async with httpx.AsyncClient(timeout=timeout) as client:
    #     headers = {"Authorization": f"Bearer {api_key}"}
    #     response = await client.get(f"{base_url}/endpoint", headers=headers)
    #     return response.json()
    
    # Example: Database operation
    # db_url = get_env_var(config.get("db_url_env", "DATABASE_URL"))
    # 
    # async with asyncpg.connect(db_url) as conn:
    #     result = await conn.fetchrow("SELECT * FROM table WHERE id = $1", message)
    #     return dict(result) if result else None
    
    # Example: File processing
    # file_path = config.get("file_path", "/tmp/data.txt")
    # max_size = config.get("max_file_size", 1024 * 1024)  # 1MB
    # 
    # async with aiofiles.open(file_path, 'r') as f:
    #     content = await f.read(max_size)
    #     return {"content": content, "size": len(content)}
    
    # Example: JSON/YAML processing
    # try:
    #     data = json.loads(message)
    #     # Process the data
    #     return {"processed": True, "data": data}
    # except json.JSONDecodeError:
    #     return {"error": "Invalid JSON format"}
    
    # Example: Multi-step workflow
    # steps = config.get("steps", [])
    # results = []
    # 
    # for step in steps:
    #     step_type = step.get("type")
    #     if step_type == "process":
    #         result = await process_step(message, step)
    #     elif step_type == "validate":
    #         result = await validate_step(message, step)
    #     else:
    #         result = {"error": f"Unknown step type: {step_type}"}
    #     
    #     results.append(result)
    # 
    # return {"workflow_results": results}


# Example: Helper function for complex tools
# async def process_step(data: str, step_config: dict) -> dict:
#     """Process a single step in a workflow."""
#     # Your step processing logic here
#     return {"step": "processed", "data": data}


# Example: Validation helper
# async def validate_step(data: str, step_config: dict) -> dict:
#     """Validate data in a workflow step."""
#     # Your validation logic here
#     return {"valid": True, "data": data}
`
}
