package frameworks

import (
	"fmt"

	"github.com/kagent-dev/kmcp/pkg/cli/internal/frameworks/golang"
	"github.com/kagent-dev/kmcp/pkg/cli/internal/frameworks/java"
	"github.com/kagent-dev/kmcp/pkg/cli/internal/frameworks/python"
	"github.com/kagent-dev/kmcp/pkg/cli/internal/frameworks/typescript"
	"github.com/kagent-dev/kmcp/pkg/cli/internal/templates"
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
	case "typescript":
		return typescript.NewGenerator(), nil
	case "java":
		return java.NewGenerator(), nil
	default:
		return nil, fmt.Errorf("unsupported framework: %s", framework)
	}
}
