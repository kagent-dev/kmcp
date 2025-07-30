package frameworks

import (
	"fmt"

	"github.com/kagent-dev/kmcp/pkg/frameworks/golang"
	"github.com/kagent-dev/kmcp/pkg/frameworks/python"
	"github.com/kagent-dev/kmcp/pkg/templates"
)

// Generator defines the interface for a framework-specific generator.
type Generator interface {
	GenerateProject(config templates.ProjectConfig) error
	GenerateTool(projectRoot string, config templates.ToolConfig) error
}

// GetGenerator returns a generator for the specified framework.
func GetGenerator(framework string) (Generator, error) {
	switch framework {
	case "fastmcp-python":
		return python.NewGenerator(), nil
	case "mcp-go":
		// TODO: Implement the Go generator.
		return golang.NewGenerator(), nil
	default:
		return nil, fmt.Errorf("unsupported framework: %s", framework)
	}
}
