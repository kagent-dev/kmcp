package golang

import (
	"fmt"

	"kagent.dev/kmcp/pkg/templates"
)

// Generator is the Go-specific generator.
type Generator struct{}

// NewGenerator creates a new Go generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateProject generates a new Go project.
func (g *Generator) GenerateProject(_ templates.ProjectConfig) error {
	return fmt.Errorf("go project generation not yet implemented")
}

// GenerateTool generates a new tool for a Go project.
func (g *Generator) GenerateTool(_ string, _ string, _ map[string]interface{}) error {
	return fmt.Errorf("go tool generation not yet implemented")
}
