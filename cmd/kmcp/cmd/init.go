package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
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
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", projectName)

	switch framework {
	case "fastmcp-python", "official-python":
		fmt.Printf("  pip install -e .\n")
		fmt.Printf("  kmcp build\n")
	case "fastmcp-ts", "easymcp-ts", "official-ts":
		fmt.Printf("  npm install\n")
		fmt.Printf("  kmcp build\n")
	}

	fmt.Printf("\nTo build a Docker image:\n")
	fmt.Printf("  kmcp build --docker\n")

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
