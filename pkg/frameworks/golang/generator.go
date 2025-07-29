package golang

import (
	"embed"
	"fmt"
	"kagent.dev/kmcp/pkg/frameworks/common"
	"kagent.dev/kmcp/pkg/templates"
	"os/exec"
)

//go:embed all:templates
var templateFiles embed.FS

// Generator is the Go-specific generator.
type Generator struct {
	common.BaseGenerator
}

// NewGenerator creates a new Go generator.
func NewGenerator() *Generator {
	return &Generator{
		BaseGenerator: common.BaseGenerator{
			TemplateFiles:    templateFiles,
			ToolTemplateName: "tool.go.tmpl",
		},
	}
}

// GenerateProject generates a new Go project.
func (g *Generator) GenerateProject(config templates.ProjectConfig) error {

	if config.Verbose {
		fmt.Println("Generating Golang MCP project...")
	}

	if err := g.BaseGenerator.GenerateProject(config); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}

	// Tidy dependencies to create go.sum
	if err := g.tidyGoMod(config.Directory, config.Verbose); err != nil {
		return fmt.Errorf("failed to finalize Go project: %w", err)
	}

	return nil
}

// initGitRepo initializes a git repository in the specified directory
func (g *Generator) initGitRepo(dir string, verbose bool) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir

	if verbose {
		fmt.Printf("  Initializing git repository...\n")
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run git init: %w", err)
	}

	return nil
}

func (g *Generator) tidyGoMod(dir string, verbose bool) error {
	if verbose {
		fmt.Println("Tidying Go module dependencies...")
	}
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if verbose && len(output) > 0 {
		fmt.Println(string(output))
	}

	if err != nil {
		return fmt.Errorf("`go mod tidy` failed: %w\n%s", err, string(output))
	}

	if verbose {
		fmt.Println("✅ Go module dependencies tidied successfully.")
	}
	return nil
}
