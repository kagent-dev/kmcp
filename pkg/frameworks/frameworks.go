package frameworks

import (
	"fmt"

	"kagent.dev/kmcp/pkg/frameworks/python"
	"kagent.dev/kmcp/pkg/templates"
)

// Generator defines the interface for a framework-specific generator.
type Generator interface {
	GenerateProject(config templates.ProjectConfig) error
	GenerateTool(projectPath string, toolName string, config map[string]interface{}) error
}

// GetGenerator returns a generator for the specified framework.
func GetGenerator(framework string) (Generator, error) {
	switch framework {
	case "fastmcp-python":
		return python.NewGenerator(), nil
	case "fastmcp-ts":
		// This will be implemented in a future step.
		// return typescript.NewGenerator(), nil
		return nil, fmt.Errorf("generator for framework '%s' not yet implemented", framework)
	default:
		return nil, fmt.Errorf("unsupported framework: %s", framework)
	}
}
