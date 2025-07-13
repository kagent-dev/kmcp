package templates

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// ProjectConfig contains configuration for generating a new project
type ProjectConfig struct {
	Name      string
	Framework string
	Template  string
	Author    string
	Email     string
	Directory string
	NoGit     bool
	Verbose   bool
}

// Generator handles project generation from templates
type Generator struct {
	// Future: Add template registry, custom template paths, etc.
}

// NewGenerator creates a new template generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateProject creates a new project from the specified configuration
func (g *Generator) GenerateProject(config ProjectConfig) error {
	if config.Verbose {
		fmt.Printf("Generating project with config: %+v\n", config)
	}

	// Create project directory
	if err := os.MkdirAll(config.Directory, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Get template data
	templateData, err := g.getTemplateData(config)
	if err != nil {
		return fmt.Errorf("failed to prepare template data: %w", err)
	}

	// Generate files based on framework and template
	switch config.Framework {
	case "fastmcp-python":
		return g.generateFastMCPPython(config, templateData)
	case "fastmcp-ts":
		return g.generateFastMCPTypeScript(config, templateData)
	case "easymcp-ts":
		return g.generateEasyMCPTypeScript(config, templateData)
	case "official-python":
		return g.generateOfficialPython(config, templateData)
	case "official-ts":
		return g.generateOfficialTypeScript(config, templateData)
	default:
		return fmt.Errorf("unsupported framework: %s", config.Framework)
	}
}

// getTemplateData prepares template variables for rendering
func (g *Generator) getTemplateData(config ProjectConfig) (map[string]interface{}, error) {
	// Convert project name to different formats
	data := map[string]interface{}{
		"ProjectName":       config.Name,
		"ProjectNameUpper":  strings.ToUpper(config.Name),
		"ProjectNameLower":  strings.ToLower(config.Name),
		"ProjectNameCamel":  g.toCamelCase(config.Name),
		"ProjectNamePascal": g.toPascalCase(config.Name),
		"ProjectNameSnake":  g.toSnakeCase(config.Name),
		"ProjectNameKebab":  g.toKebabCase(config.Name),
		"Framework":         config.Framework,
		"Template":          config.Template,
		"Author":            config.Author,
		"Email":             config.Email,
		"Year":              "2025", // You could make this dynamic
	}

	// Set default author if empty
	if data["Author"] == "" {
		data["Author"] = "KMCP CLI"
	}
	if data["Email"] == "" {
		data["Email"] = "noreply@kagent.dev"
	}

	return data, nil
}

// generateFastMCPPython creates a FastMCP Python project
func (g *Generator) generateFastMCPPython(config ProjectConfig, data map[string]interface{}) error {
	if config.Verbose {
		fmt.Println("Generating FastMCP Python project...")
	}

	// Create basic directory structure
	dirs := []string{
		"",
		g.toSnakeCase(config.Name),
		"tests",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(config.Directory, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
	}

	// Generate files
	files := g.getFastMCPPythonFiles(config.Template, data)

	for filename, content := range files {
		filePath := filepath.Join(config.Directory, filename)

		// Ensure directory exists for the file
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for file %s: %w", filename, err)
		}

		// Render template content
		renderedContent, err := g.renderTemplate(content, data)
		if err != nil {
			return fmt.Errorf("failed to render template for %s: %w", filename, err)
		}

		// Write file
		if err := os.WriteFile(filePath, []byte(renderedContent), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filename, err)
		}

		if config.Verbose {
			fmt.Printf("  Created: %s\n", filename)
		}
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

// renderTemplate renders a template string with the provided data
func (g *Generator) renderTemplate(tmplContent string, data map[string]interface{}) (string, error) {
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
		result += strings.Title(strings.ToLower(word))
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

func (g *Generator) toSnakeCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	return strings.ToLower(strings.Join(words, "_"))
}

func (g *Generator) toKebabCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	return strings.ToLower(strings.Join(words, "-"))
}

// Placeholder methods for other frameworks - will implement incrementally
func (g *Generator) generateFastMCPTypeScript(config ProjectConfig, data map[string]interface{}) error {
	return fmt.Errorf("FastMCP TypeScript template not yet implemented")
}

func (g *Generator) generateEasyMCPTypeScript(config ProjectConfig, data map[string]interface{}) error {
	return fmt.Errorf("EasyMCP TypeScript template not yet implemented")
}

func (g *Generator) generateOfficialPython(config ProjectConfig, data map[string]interface{}) error {
	return fmt.Errorf("Official Python SDK template not yet implemented")
}

func (g *Generator) generateOfficialTypeScript(config ProjectConfig, data map[string]interface{}) error {
	return fmt.Errorf("Official TypeScript SDK template not yet implemented")
}
