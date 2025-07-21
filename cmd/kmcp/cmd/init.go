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

const (
	frameworkFastMCPPython = "fastmcp-python"
	frameworkFastMCPTS     = "fastmcp-ts"
	templateBasic          = "basic"
	templateDatabase       = "database"
	templateFilesystem     = "filesystem"
	templateAPIClient      = "api-client"
	templateMultiTool      = "multi-tool"
	templateWorkflow       = "workflow"
	templateData           = "data"
	templateHTTP           = "http"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new MCP server project",
	Long: `Initialize a new MCP server project with dynamic tool loading.

This command creates a new MCP server project using one of the supported frameworks:
- FastMCP Python (recommended) - Dynamic tool loading with FastMCP
- FastMCP TypeScript - Dynamic tool loading with FastMCP  

The recommended approach is FastMCP Python which provides:
- Automatic tool discovery and loading
- One file per tool with clear structure
- Built-in configuration management
- Comprehensive testing framework`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

var (
	initFramework      string
	initForce          bool
	initNoGit          bool
	initAuthor         string
	initEmail          string
	initVersion        string
	initNonInteractive bool
)

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&initFramework, "framework", "f", "", "Framework to use (fastmcp-python, fastmcp-ts)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing directory")
	initCmd.Flags().BoolVar(&initNoGit, "no-git", false, "Skip git initialization")
	initCmd.Flags().StringVar(&initAuthor, "author", "", "Author name for the project")
	initCmd.Flags().StringVar(&initEmail, "email", "", "Author email for the project")
	initCmd.Flags().StringVar(&initVersion, "version", "", "MCP server version (defaults to kmcp version)")
	initCmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false, "Run in non-interactive mode")
}

func runInit(_ *cobra.Command, args []string) error {
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
	template := "basic" // Default template for all frameworks
	if !initNonInteractive && framework == "fastmcp-python" {
		fmt.Println("\nFastMCP Python uses dynamic tool loading - no template selection needed!")
		fmt.Println("Tools will be automatically discovered from the src/tools/ directory.")
		fmt.Println("Use 'kmcp add-tool <name>' to add new tools after project creation.")
	} else if !initNonInteractive {
		selected, err := promptForTemplate(framework)
		if err != nil {
			return fmt.Errorf("failed to select template: %w", err)
		}
		template = selected
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
	case frameworkFastMCPPython:
		fmt.Printf("  uv sync\n")
		fmt.Printf("  uv run python src/main.py\n")
	case frameworkFastMCPTS:
		fmt.Printf("  npm install\n")
		fmt.Printf("  kmcp build\n")
	}

	fmt.Printf("\nTo build a Docker image:\n")
	fmt.Printf("  kmcp build --docker\n")

	fmt.Printf("\nTo build a Docker image in a specific directory:\n")
	fmt.Printf("  kmcp build --docker --dir ./my-project\n")

	fmt.Printf("\nTo develop using Docker only (no local Python/uv required):\n")
	fmt.Printf("  kmcp build --docker --verbose  # Build and test\n")
	fmt.Printf("  kmcp deploy --apply           # Deploy to Kubernetes\n")

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
	fmt.Println("1. FastMCP Python (recommended) - Dynamic tool loading with FastMCP")
	fmt.Println("2. FastMCP TypeScript - Dynamic tool loading with FastMCP")
	fmt.Print("Enter choice [1-2]: ")

	var choice string
	if _, err := fmt.Scanln(&choice); err != nil {
		return "", err
	}

	switch strings.TrimSpace(choice) {
	case "1", "":
		return frameworkFastMCPPython, nil
	case "2":
		return frameworkFastMCPTS, nil
	default:
		return frameworkFastMCPPython, nil // Default to recommended
	}
}

func promptForTemplate(framework string) (string, error) {
	if framework == frameworkFastMCPPython {
		// FastMCP Python uses dynamic loading, no template selection needed
		return templateBasic, nil
	}

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
		return templateBasic, nil
	case "2":
		return templateDatabase, nil
	case "3":
		return templateFilesystem, nil
	case "4":
		return templateAPIClient, nil
	case "5":
		return templateMultiTool, nil
	default:
		return templateBasic, nil // Default to basic
	}
}

func promptForAuthor() (string, error) {
	fmt.Print("Enter author name (optional): ")
	var author string
	_, err := fmt.Scanln(&author)
	if err != nil {
		return "", fmt.Errorf("failed to read author: %w", err)
	}
	return strings.TrimSpace(author), nil
}

func promptForEmail() (string, error) {
	fmt.Print("Enter author email (optional): ")
	var email string
	_, err := fmt.Scanln(&email)
	if err != nil {
		return "", fmt.Errorf("failed to read email: %w", err)
	}
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

	version := initVersion
	if version == "" {
		version = Version
	}

	// Create manifest with template-specific tools
	projectManifest := &manifest.ProjectManifest{
		Name:        projectName,
		Framework:   framework,
		Version:     version,
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
	case templateBasic:
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
	case templateHTTP:
		tools["http_client"] = manifest.ToolConfig{
			Name:        "http_client",
			Description: "Make HTTP requests",
			Handler:     "tools.http_client",
			Config: map[string]interface{}{
				"enabled": true,
				"timeout": 30,
			},
		}
	case templateData:
		tools["data_processor"] = manifest.ToolConfig{
			Name:        "data_processor",
			Description: "Process and manipulate data",
			Handler:     "tools.data_processor",
			Config: map[string]interface{}{
				"enabled": true,
			},
		}
	case templateWorkflow:
		tools["workflow_executor"] = manifest.ToolConfig{
			Name:        "workflow_executor",
			Description: "Execute multi-step workflows",
			Handler:     "tools.workflow_executor",
			Config: map[string]interface{}{
				"enabled":   true,
				"max_steps": 10,
			},
		}
	case templateMultiTool:
		// Combine all tools
		for k, v := range getTemplateTools(templateBasic) {
			tools[k] = v
		}
		for k, v := range getTemplateTools(templateHTTP) {
			tools[k] = v
		}
		for k, v := range getTemplateTools(templateData) {
			tools[k] = v
		}
		for k, v := range getTemplateTools(templateWorkflow) {
			tools[k] = v
		}
	default:
		// Default to basic tools
		return getTemplateTools(templateBasic)
	}

	return tools
}

// getFrameworkDependencies returns runtime dependencies for a framework
func getFrameworkDependencies(framework, template string) []string {
	switch framework {
	case frameworkFastMCPPython:
		deps := []string{"mcp>=1.0.0", "fastmcp>=0.1.0"}
		switch template {
		case templateHTTP:
			deps = append(deps, "httpx>=0.25.0")
		case templateData:
			deps = append(deps, "pandas>=2.0.0", "numpy>=1.21.0")
		case templateWorkflow:
			deps = append(deps, "asyncio")
		}
		return deps
	case frameworkFastMCPTS:
		deps := []string{"@fastmcp/core", "@modelcontextprotocol/sdk"}
		switch template {
		case templateHTTP:
			deps = append(deps, "axios", "fetch")
		case templateData:
			deps = append(deps, "lodash")
		case templateWorkflow:
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
	case frameworkFastMCPPython:
		return []string{"pytest>=7.0.0", "pytest-asyncio>=0.21.0", "black>=22.0.0", "mypy>=1.0.0", "ruff>=0.1.0"}
	case frameworkFastMCPTS:
		return []string{"@types/node", "typescript", "tsx", "vitest", "eslint", "prettier"}
	default:
		return []string{}
	}
}
