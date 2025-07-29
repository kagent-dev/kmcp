package golang

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"kagent.dev/kmcp/pkg/templates"
)

//go:embed all:templates
var templateFiles embed.FS

// Generator is the Go-specific generator.
type Generator struct{}

// NewGenerator creates a new Go generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateProject generates a new Go project.
func (g *Generator) GenerateProject(config templates.ProjectConfig) error {
	// Create project directory
	if err := os.MkdirAll(config.Directory, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Define templates to render
	templateMap := map[string]string{
		"go.mod.tmpl":             "go.mod",
		"main.go.tmpl":            "main.go",
		"tools/all_tools.go.tmpl": "tools/all_tools.go",
		"tools/echo.go.tmpl":      "tools/echo.go",
		"Dockerfile.tmpl":         "Dockerfile",
		".gitignore.tmpl":         ".gitignore",
		"README.md.tmpl":          "README.md",
		"kmcp.yaml.tmpl":          "kmcp.yaml",
	}

	for tmplName, outputName := range templateMap {
		// Create parent directory if it doesn't exist
		outputDir := filepath.Dir(filepath.Join(config.Directory, outputName))
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", outputName, err)
		}

		// Generate file from template
		if err := g.generateFileFromTemplate(config.Directory, tmplName, outputName, config); err != nil {
			return fmt.Errorf("failed to generate %s: %w", outputName, err)
		}
	}

	// Tidy dependencies to create go.sum
	if err := g.tidyGoMod(config.Directory, config.Verbose); err != nil {
		return fmt.Errorf("failed to finalize Go project: %w", err)
	}

	// Initialize git repository
	if !config.NoGit {
		if err := g.initGit(config.Directory); err != nil {
			return fmt.Errorf("failed to initialize git repository: %w", err)
		}
	}

	return nil
}

// GenerateTool generates a new tool for a Go project.
func (g *Generator) GenerateTool(projectPath string, toolName string, config map[string]interface{}) error {
	// Prepare template data
	data := map[string]interface{}{
		"ToolName": toolName,
	}
	for key, value := range config {
		data[key] = value
	}

	// Generate file from template
	if err := g.generateFileFromTemplate(projectPath, "tool.go.tmpl", "tools/"+toolName+".go", data); err != nil {
		return fmt.Errorf("failed to generate tool file: %w", err)
	}

	fmt.Printf("✅ Successfully created tool: %s\n", toolName)
	fmt.Printf("📁 Generated file: tools/%s.go\n", toolName)
	fmt.Printf("🔵 Remember to add the new tool to main.go\n")

	return nil
}

func (g *Generator) generateFileFromTemplate(projectDir, templateName, outputName string, data interface{}) error {
	templatePath := filepath.Join("templates", templateName)
	outputFilePath := filepath.Join(projectDir, outputName)

	// Read template content
	templateContent, err := templateFiles.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templateName, err)
	}

	// Parse template
	tmpl, err := template.New(templateName).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	// Create output file
	file, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputFilePath, err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return nil
}

func (g *Generator) initGit(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
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
