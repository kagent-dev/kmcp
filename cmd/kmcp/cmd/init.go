package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"kagent.dev/kmcp/pkg/frameworks"
	"kagent.dev/kmcp/pkg/manifest"
	"kagent.dev/kmcp/pkg/templates"
)

const (
	frameworkFastMCPPython = "fastmcp-python"
	frameworkMCPGo         = "mcp-go"
	templateBasic          = "basic"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new MCP server project",
	Long: `Initialize a new MCP server project with dynamic tool loading.

This command creates a new MCP server project using one of the supported frameworks:
- FastMCP Python (recommended) - Dynamic tool loading with FastMCP

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
	initDescription    string
	initNonInteractive bool
)

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&initFramework, "framework", "f", "", "Framework to use (fastmcp-python)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing directory")
	initCmd.Flags().BoolVar(&initNoGit, "no-git", false, "Skip git initialization")
	initCmd.Flags().StringVar(&initAuthor, "author", "", "Author name for the project")
	initCmd.Flags().StringVar(&initEmail, "email", "", "Author email for the project")
	initCmd.Flags().StringVar(&initDescription, "description", "", "Description for the project")
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
		framework = promptForFramework()
	} else if framework == "" {
		framework = frameworkFastMCPPython // Default framework
	}

	// Get author information
	author := initAuthor
	email := initEmail
	description := initDescription
	if !initNonInteractive {
		if author == "" {
			author = promptForAuthor()
		}
		if email == "" {
			email = promptForEmail()
		}
		if description == "" {
			description = promptForDescription()
		}
	}

	if verbose {
		fmt.Printf("Creating %s project using %s framework\n", projectName, framework)
		fmt.Printf("Project directory: %s\n", projectPath)
	}

	// Create project directory first
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create the project manifest object
	projectManifest := manifest.GetDefault(projectName, framework, description, author, email)

	// Create project configuration
	projectConfig := templates.ProjectConfig{
		ProjectName: projectManifest.Name,
		Framework:   projectManifest.Framework,
		Version:     projectManifest.Version,
		Description: projectManifest.Description,
		Author:      projectManifest.Author,
		Email:       projectManifest.Email,
		Tools:       projectManifest.Tools,
		Secrets:     projectManifest.Secrets,
		Build:       projectManifest.Build,
		Directory:   projectPath,
		NoGit:       initNoGit,
		Verbose:     verbose,
	}

	// Use the generator to create the project files
	fmt.Println("✅ Creating project structure...")
	generator, err := frameworks.GetGenerator(framework)
	if err != nil {
		return fmt.Errorf("failed to get generator: %w", err)
	}

	if err := generator.GenerateProject(projectConfig); err != nil {
		return fmt.Errorf("failed to generate project files: %w", err)
	}

	// Create kmcp.yaml
	manifestManager := manifest.NewManager(projectPath)
	if err := manifestManager.Save(projectManifest); err != nil {
		return fmt.Errorf("failed to save project manifest: %w", err)
	}

	// Get absolute path for output
	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		absProjectPath = projectPath // Fallback to relative path if absolute fails
	}

	// Success message
	fmt.Printf("\n✓ Successfully created MCP server project: %s\n", projectName)
	fmt.Printf("✓ Generated project manifest: kmcp.yaml\n")
	fmt.Printf("\nNext steps for local development:\n")

	switch framework {
	case frameworkFastMCPPython:
		fmt.Printf("  Tools will be automatically discovered from the src/tools/ directory.\n")
		fmt.Printf("  Use 'kmcp add-tool <name>' to add new tools after project creation.\n")
		fmt.Printf("\n")
		fmt.Printf("  To connect to the server using the inspector:\n")
		fmt.Printf("  run npx @modelcontextprotocol/inspector\n")
		fmt.Printf("  open the inspector on localhost:6274 and set transport type to STDIO\n")
		fmt.Printf("  copy the `MCP_PROXY_AUTH_TOKEN` into the Proxy Session Token input under configuration\n")
		fmt.Printf("  paste the following command into the inspector to connect to the server using the inspector\n")
		fmt.Printf("  %s\n", filepath.Join(absProjectPath, "run_server.sh"))
		fmt.Printf("\n")
		fmt.Printf("  alternatively, run the following commands to start the server\n")
		fmt.Printf("  cd %s\n", projectName)
		fmt.Printf("  uv sync\n")
		fmt.Printf("  uv run python src/main.py\n")
	case frameworkMCPGo:
		fmt.Printf("  go mod tidy\n")
		fmt.Printf("  go run main.go\n")
	}

	fmt.Printf("\nTo build a Docker image:\n")
	fmt.Printf("  kmcp build --docker\n")

	fmt.Printf("\nTo build a Docker image in a specific directory:\n")
	fmt.Printf("  kmcp build --docker --dir ./my-project\n")

	fmt.Printf("\nTo develop using Docker only (no local Python/uv required):\n")
	fmt.Printf("  kmcp build --docker --verbose  # Build and test\n")
	fmt.Printf("  kmcp deploy mcp --apply       # Deploy MCP server to Kubernetes\n")

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

// Prompts for user input

func promptForProjectName() (string, error) {
	fmt.Print("Enter project name: ")
	var name string
	if _, err := fmt.Scanln(&name); err != nil {
		return "", err
	}
	return strings.TrimSpace(name), nil
}

func promptForFramework() string {
	fmt.Println("\nSelect a framework:")
	fmt.Println("1. FastMCP Python (recommended) - Dynamic tool loading with FastMCP")
	fmt.Print("Enter choice [1]: ")

	var choice string
	if _, err := fmt.Scanln(&choice); err != nil {
		// Default to FastMCP Python on any scan error (e.g., empty input)
		return frameworkFastMCPPython
	}

	switch strings.TrimSpace(choice) {
	case "1", "":
		return frameworkFastMCPPython
	case "2":
		return frameworkMCPGo
	default:
		return frameworkFastMCPPython // Default to recommended
	}
}

func promptForAuthor() string {
	fmt.Print("Enter author name (optional): ")
	var author string
	if _, err := fmt.Scanln(&author); err != nil {
		return ""
	}
	return strings.TrimSpace(author)
}

func promptForEmail() string {
	fmt.Print("Enter author email (optional): ")
	var email string
	if _, err := fmt.Scanln(&email); err != nil {
		return ""
	}
	return strings.TrimSpace(email)
}

func promptForDescription() string {
	fmt.Print("Enter description (optional): ")
	var description string
	if _, err := fmt.Scanln(&description); err != nil {
		return "" // Ignore error, treat as empty
	}
	return strings.TrimSpace(description)
}
