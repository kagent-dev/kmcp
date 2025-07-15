package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"kagent.dev/kmcp/pkg/manifest"
	"kagent.dev/kmcp/pkg/templates"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new MCP server project",
	Long: `Initialize a new MCP server project with templates and best practices.

This command creates a new MCP server project using one of the supported frameworks:
- FastMCP Python (recommended)
- FastMCP TypeScript  
- EasyMCP TypeScript (simple)
- Official Python SDK
- Official TypeScript SDK

The command will guide you through selecting a framework and template type.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

var (
	initFramework      string
	initTemplate       string
	initForce          bool
	initNoGit          bool
	initAuthor         string
	initEmail          string
	initNonInteractive bool
)

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&initFramework, "framework", "f", "", "Framework to use (fastmcp-python, fastmcp-ts, easymcp-ts, official-python, official-ts)")
	initCmd.Flags().StringVarP(&initTemplate, "template", "t", "", "Template type (basic, database, filesystem, api-client, multi-tool)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing directory")
	initCmd.Flags().BoolVar(&initNoGit, "no-git", false, "Skip git initialization")
	initCmd.Flags().StringVar(&initAuthor, "author", "", "Author name for the project")
	initCmd.Flags().StringVar(&initEmail, "email", "", "Author email for the project")
	initCmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false, "Run in non-interactive mode")
}

func runInit(cmd *cobra.Command, args []string) error {
	var projectName string

	// Get project name from args or prompt
	if len(args) > 0 {
		projectName = args[0]
	} else if !initNonInteractive {
		name, err := promptForProjectName()
		if err != nil {
			return fmt.Errorf("failed to get project name: %w", err)
		}
		projectName = name
	} else {
		return fmt.Errorf("project name is required in non-interactive mode")
	}

	// Validate project name
	if err := validateProjectName(projectName); err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}

	// Check if directory exists
	projectPath := filepath.Join(".", projectName)
	if _, err := os.Stat(projectPath); err == nil && !initForce {
		return fmt.Errorf("directory '%s' already exists. Use --force to overwrite", projectName)
	}

	// Get framework selection
	framework := initFramework
	if framework == "" && !initNonInteractive {
		selected, err := promptForFramework()
		if err != nil {
			return fmt.Errorf("failed to select framework: %w", err)
		}
		framework = selected
	} else if framework == "" {
		framework = "fastmcp-python" // Default framework
	}

	// Get template selection
	template := initTemplate
	if template == "" && !initNonInteractive {
		selected, err := promptForTemplate(framework)
		if err != nil {
			return fmt.Errorf("failed to select template: %w", err)
		}
		template = selected
	} else if template == "" {
		template = "basic" // Default template
	}

	// Get author information
	author := initAuthor
	email := initEmail
	if !initNonInteractive {
		if author == "" {
			author, _ = promptForAuthor()
		}
		if email == "" {
			email, _ = promptForEmail()
		}
	}

	if verbose {
		fmt.Printf("Creating %s project using %s framework with %s template\n", projectName, framework, template)
		fmt.Printf("Project directory: %s\n", projectPath)
	}

	// Create project directory first
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create project manifest
	if err := createProjectManifest(projectPath, projectName, framework, template, author, email); err != nil {
		return fmt.Errorf("failed to create project manifest: %w", err)
	}

	// Create project configuration
	config := templates.ProjectConfig{
		Name:      projectName,
		Framework: framework,
		Template:  template,
		Author:    author,
		Email:     email,
		Directory: projectPath,
		NoGit:     initNoGit,
		Verbose:   verbose,
	}

	// Initialize the project
	generator := templates.NewGenerator()
	if err := generator.GenerateProject(config); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}

	// Success message
	fmt.Printf("\n✓ Successfully created MCP server project: %s\n", projectName)
	fmt.Printf("✓ Generated project manifest: kmcp.yaml\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", projectName)

	switch framework {
	case "fastmcp-python", "official-python":
		fmt.Printf("  uv sync\n")
		fmt.Printf("  uv run python -m src.main\n")
	case "fastmcp-ts", "easymcp-ts", "official-ts":
		fmt.Printf("  npm install\n")
		fmt.Printf("  kmcp build\n")
	}

	fmt.Printf("\nTo build a Docker image:\n")
	fmt.Printf("  kmcp build --docker\n")

	fmt.Printf("\nTo manage secrets:\n")
	fmt.Printf("  kmcp secrets add-secret API_KEY --environment local\n")
	fmt.Printf("  kmcp secrets generate-k8s-secrets --environment staging\n")

	return nil
}

func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// Check for invalid characters
	if strings.ContainsAny(name, " \t\n\r/\\:*?\"<>|") {
		return fmt.Errorf("project name contains invalid characters")
	}

	// Check if it starts with a dot
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("project name cannot start with a dot")
	}

	return nil
}

// Simple prompts for now - we'll enhance these with better UX later
func promptForProjectName() (string, error) {
	fmt.Print("Enter project name: ")
	var name string
	if _, err := fmt.Scanln(&name); err != nil {
		return "", err
	}
	return strings.TrimSpace(name), nil
}

func promptForFramework() (string, error) {
	fmt.Println("\nSelect a framework:")
	fmt.Println("1. FastMCP Python (recommended)")
	fmt.Println("2. FastMCP TypeScript")
	fmt.Println("3. EasyMCP TypeScript (simple)")
	fmt.Println("4. Official Python SDK")
	fmt.Println("5. Official TypeScript SDK")
	fmt.Print("Enter choice [1-5]: ")

	var choice string
	if _, err := fmt.Scanln(&choice); err != nil {
		return "", err
	}

	switch strings.TrimSpace(choice) {
	case "1", "":
		return "fastmcp-python", nil
	case "2":
		return "fastmcp-ts", nil
	case "3":
		return "easymcp-ts", nil
	case "4":
		return "official-python", nil
	case "5":
		return "official-ts", nil
	default:
		return "fastmcp-python", nil // Default to recommended
	}
}

func promptForTemplate(framework string) (string, error) {
	fmt.Println("\nSelect a template:")
	fmt.Println("1. Basic - Simple server with example tools")
	fmt.Println("2. Database - Database integration template")
	fmt.Println("3. Filesystem - File system access template")
	fmt.Println("4. API Client - REST/GraphQL API integration")
	fmt.Println("5. Multi-tool - Comprehensive server with multiple capabilities")
	fmt.Print("Enter choice [1-5]: ")

	var choice string
	if _, err := fmt.Scanln(&choice); err != nil {
		return "", err
	}

	switch strings.TrimSpace(choice) {
	case "1", "":
		return "basic", nil
	case "2":
		return "database", nil
	case "3":
		return "filesystem", nil
	case "4":
		return "api-client", nil
	case "5":
		return "multi-tool", nil
	default:
		return "basic", nil // Default to basic
	}
}

func promptForAuthor() (string, error) {
	fmt.Print("Enter author name (optional): ")
	var author string
	fmt.Scanln(&author)
	return strings.TrimSpace(author), nil
}

func promptForEmail() (string, error) {
	fmt.Print("Enter author email (optional): ")
	var email string
	fmt.Scanln(&email)
	return strings.TrimSpace(email), nil
}

// createProjectManifest creates the kmcp.yaml manifest file
func createProjectManifest(projectPath, projectName, framework, template, author, email string) error {
	// Set default author if empty
	if author == "" {
		author = "KMCP CLI"
	}
	if email == "" {
		email = "noreply@kagent.dev"
	}

	// Create manifest with template-specific tools
	projectManifest := &manifest.ProjectManifest{
		Name:        projectName,
		Framework:   framework,
		Version:     "0.1.0",
		Description: fmt.Sprintf("%s MCP server built with %s", projectName, framework),
		Author:      author,
		Email:       email,
		Tools:       getTemplateTools(template),
		Resources:   make(map[string]manifest.ResourceConfig),
		Secrets: manifest.SecretsConfig{
			Local: manifest.SecretProviderConfig{
				Provider: manifest.SecretProviderEnv,
				Config: map[string]interface{}{
					"file": ".env.local",
				},
			},
			Staging: manifest.SecretProviderConfig{
				Provider: manifest.SecretProviderKubernetes,
				Config: map[string]interface{}{
					"namespace":  "default",
					"secretName": fmt.Sprintf("%s-secrets-staging", strings.ReplaceAll(projectName, "_", "-")),
				},
			},
			Production: manifest.SecretProviderConfig{
				Provider: manifest.SecretProviderKubernetes,
				Config: map[string]interface{}{
					"namespace":  "production",
					"secretName": fmt.Sprintf("%s-secrets-production", strings.ReplaceAll(projectName, "_", "-")),
				},
			},
		},
		Dependencies: manifest.DependencyConfig{
			Runtime: getFrameworkDependencies(framework, template),
			Dev:     getFrameworkDevDependencies(framework),
		},
		Build: manifest.BuildConfig{
			Output: projectName,
			Docker: manifest.DockerConfig{
				Image:      fmt.Sprintf("%s:latest", strings.ReplaceAll(projectName, "_", "-")),
				Dockerfile: "Dockerfile",
				Platform:   []string{"linux/amd64"},
			},
		},
	}

	// Save manifest
	manifestManager := manifest.NewManager(projectPath)
	if err := manifestManager.Save(projectManifest); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	return nil
}

// getTemplateTools returns the tools configuration for a given template
func getTemplateTools(template string) map[string]manifest.ToolConfig {
	tools := make(map[string]manifest.ToolConfig)

	switch template {
	case "basic":
		tools["echo"] = manifest.ToolConfig{
			Name:        "echo",
			Description: "Echo a message back to the client",
			Handler:     "tools.echo",
			Config: map[string]interface{}{
				"enabled": true,
			},
		}
		tools["calculator"] = manifest.ToolConfig{
			Name:        "calculator",
			Description: "Perform basic arithmetic calculations",
			Handler:     "tools.calculator",
			Config: map[string]interface{}{
				"enabled": true,
			},
		}
	case "http":
		tools["http_client"] = manifest.ToolConfig{
			Name:        "http_client",
			Description: "Make HTTP requests",
			Handler:     "tools.http_client",
			Config: map[string]interface{}{
				"enabled": true,
				"timeout": 30,
			},
		}
	case "data":
		tools["data_processor"] = manifest.ToolConfig{
			Name:        "data_processor",
			Description: "Process and manipulate data",
			Handler:     "tools.data_processor",
			Config: map[string]interface{}{
				"enabled": true,
			},
		}
	case "workflow":
		tools["workflow_executor"] = manifest.ToolConfig{
			Name:        "workflow_executor",
			Description: "Execute multi-step workflows",
			Handler:     "tools.workflow_executor",
			Config: map[string]interface{}{
				"enabled":   true,
				"max_steps": 10,
			},
		}
	case "multi-tool":
		// Combine all tools
		for k, v := range getTemplateTools("basic") {
			tools[k] = v
		}
		for k, v := range getTemplateTools("http") {
			tools[k] = v
		}
		for k, v := range getTemplateTools("data") {
			tools[k] = v
		}
		for k, v := range getTemplateTools("workflow") {
			tools[k] = v
		}
	default:
		// Default to basic tools
		return getTemplateTools("basic")
	}

	return tools
}

// getFrameworkDependencies returns runtime dependencies for a framework
func getFrameworkDependencies(framework, template string) []string {
	switch framework {
	case "fastmcp-python":
		deps := []string{"mcp>=1.0.0", "fastmcp>=0.1.0"}
		switch template {
		case "http":
			deps = append(deps, "httpx>=0.25.0")
		case "data":
			deps = append(deps, "pandas>=2.0.0", "numpy>=1.21.0")
		case "workflow":
			deps = append(deps, "asyncio")
		}
		return deps
	case "fastmcp-ts":
		deps := []string{"@fastmcp/core", "@modelcontextprotocol/sdk"}
		switch template {
		case "http":
			deps = append(deps, "axios", "fetch")
		case "data":
			deps = append(deps, "lodash")
		case "workflow":
			deps = append(deps, "async")
		}
		return deps
	case "official-python":
		deps := []string{"mcp>=1.0.0"}
		switch template {
		case "http":
			deps = append(deps, "httpx>=0.25.0")
		case "data":
			deps = append(deps, "pandas>=2.0.0")
		case "workflow":
			deps = append(deps, "asyncio")
		}
		return deps
	case "official-typescript":
		deps := []string{"@modelcontextprotocol/sdk"}
		switch template {
		case "http":
			deps = append(deps, "axios")
		case "data":
			deps = append(deps, "lodash")
		case "workflow":
			deps = append(deps, "async")
		}
		return deps
	default:
		return []string{}
	}
}

// getFrameworkDevDependencies returns development dependencies for a framework
func getFrameworkDevDependencies(framework string) []string {
	switch framework {
	case "fastmcp-python", "official-python":
		return []string{"pytest>=7.0.0", "pytest-asyncio>=0.21.0", "black>=22.0.0", "mypy>=1.0.0", "ruff>=0.1.0"}
	case "fastmcp-ts", "easymcp-ts", "official-ts":
		return []string{"@types/node", "typescript", "tsx", "vitest", "eslint", "prettier"}
	default:
		return []string{}
	}
}
