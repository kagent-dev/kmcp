package python

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"kagent.dev/kmcp/pkg/templates"
)

//go:embed all:templates
var templateFiles embed.FS

// Generator for Python projects
type Generator struct{}

// NewGenerator creates a new Python generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateProject generates a new Python project
func (g *Generator) GenerateProject(config templates.ProjectConfig) error {
	if config.Framework == "fastmcp-python" {
		// Generate project from embedded templates
		return g.generateFastMCPPython(config)
	}
	return fmt.Errorf("unsupported python framework: %s", config.Framework)
}

// GenerateTool generates a new tool for a Python project.
func (g *Generator) GenerateTool(projectPath string, toolName string, config map[string]interface{}) error {
	toolPath := filepath.Join(projectPath, "src", "tools", toolName+".py")
	if err := g.GenerateToolFile(toolPath, toolName, config); err != nil {
		return fmt.Errorf("failed to generate tool file: %w", err)
	}

	// After generating the tool file, regenerate the __init__.py file
	toolsDir := filepath.Dir(toolPath)
	if err := g.RegenerateToolsInit(toolsDir); err != nil {
		return fmt.Errorf("failed to regenerate __init__.py: %w", err)
	}
	return nil
}

func (g *Generator) generateFastMCPPython(config templates.ProjectConfig) error {
	if config.Verbose {
		fmt.Println("Generating FastMCP Python project...")
	}

	templateRoot, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		return fmt.Errorf("failed to get templates subdirectory: %w", err)
	}

	err = fs.WalkDir(templateRoot, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		destPath := filepath.Join(config.Directory, strings.TrimSuffix(path, ".tmpl"))

		if d.IsDir() {
			// Create the directory if it doesn't exist
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			return nil
		}

		// Read template file
		templateContent, err := fs.ReadFile(templateRoot, path)
		if err != nil {
			return fmt.Errorf("failed to read template file %s: %w", path, err)
		}

		// Render template content
		renderedContent, err := g.renderTemplate(string(templateContent), config)
		if err != nil {
			return fmt.Errorf("failed to render template for %s: %w", path, err)
		}

		// Create file
		if err := os.WriteFile(destPath, []byte(renderedContent), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", destPath, err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk templates: %w", err)
	}

	// Initialize git repository
	if !config.NoGit {
		if err := g.initGitRepo(config.Directory, config.Verbose); err != nil {
			// Don't fail the whole operation if git init fails
			if config.Verbose {
				fmt.Printf("Warning: failed to initialize git repository: %v\n", err)
			}
		}
	}

	return nil
}

// GenerateToolFile generates a new Python tool file from the unified template
func (g *Generator) GenerateToolFile(filePath, toolName string, config map[string]interface{}) error {
	// Prepare template data
	data := map[string]interface{}{
		"ToolName":      toolName,
		"ToolNameTitle": cases.Title(language.English).String(toolName),
		"ToolNameUpper": strings.ToUpper(toolName),
		"ToolNameLower": strings.ToLower(toolName),
		"ClassName":     cases.Title(language.English).String(toolName) + "Tool",
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
	templateContent, err := fs.ReadFile(templateFiles, "templates/tool.py.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read tool template: %w", err)
	}

	tmpl, err := template.New("tool").Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create the output file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	// Execute the template
	err = tmpl.Execute(file, data)

	// Close the file and check for errors
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

// RegenerateToolsInit regenerates the __init__.py file in the tools directory
func (g *Generator) RegenerateToolsInit(toolsDir string) error {
	// Scan the tools directory for Python files
	tools, err := g.ScanToolsDirectory(toolsDir)
	if err != nil {
		return fmt.Errorf("failed to scan tools directory: %w", err)
	}

	// Generate the __init__.py content
	content := g.generateInitContent(tools)

	// Write the __init__.py file
	initPath := filepath.Join(toolsDir, "__init__.py")
	return os.WriteFile(initPath, []byte(content), 0644)
}

// ScanToolsDirectory scans the tools directory and returns a list of tool names
func (g *Generator) ScanToolsDirectory(toolsDir string) ([]string, error) {
	var tools []string

	// Read the directory
	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tools directory: %w", err)
	}

	// Find all Python files (excluding __init__.py)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".py") && name != "__init__.py" {
			// Extract tool name (filename without .py extension)
			toolName := strings.TrimSuffix(name, ".py")
			tools = append(tools, toolName)
		}
	}

	return tools, nil
}

// generateInitContent generates the content for the __init__.py file
func (g *Generator) generateInitContent(tools []string) string {
	var content strings.Builder

	// Add the header comment
	content.WriteString(`"""Tools package for knowledge-assistant MCP server.

This file is automatically generated by the dynamic loading system.
Do not edit manually - it will be overwritten when tools are loaded.
"""

`)

	// Add import statements
	for _, tool := range tools {
		content.WriteString(fmt.Sprintf("from .%s import %s\n", tool, tool))
	}

	// Add empty line
	content.WriteString("\n")

	// Add __all__ list
	content.WriteString("__all__ = [")
	for i, tool := range tools {
		if i > 0 {
			content.WriteString(", ")
		}
		content.WriteString(fmt.Sprintf(`"%s"`, tool))
	}
	content.WriteString("]\n")

	return content.String()
}

// renderTemplate renders a template string with the provided data
func (g *Generator) renderTemplate(tmplContent string, data interface{}) (string, error) {
	tmpl, err := template.New("template").Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return result.String(), nil
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

// String manipulation utilities
func (g *Generator) toCamelCase(s string) string {
	if s == "" {
		return s
	}
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	if len(words) == 0 {
		return s
	}
	result := strings.ToLower(words[0])
	for _, word := range words[1:] {
		result += cases.Title(language.English).String(strings.ToLower(word))
	}
	return result
}

func (g *Generator) toPascalCase(s string) string {
	camel := g.toCamelCase(s)
	if camel == "" {
		return camel
	}
	return strings.ToUpper(camel[:1]) + camel[1:]
}

func (g *Generator) toKebabCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	return strings.ToLower(strings.Join(words, "-"))
}
