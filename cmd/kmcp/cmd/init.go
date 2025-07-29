package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"kagent.dev/kmcp/pkg/frameworks"
	"kagent.dev/kmcp/pkg/manifest"
	"kagent.dev/kmcp/pkg/templates"

	"github.com/spf13/cobra"
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

This command provides subcommands to initialize a new MCP server project
using one of the supported frameworks.`,
	RunE: runInit,
}

var (
	initForce          bool
	initNoGit          bool
	initAuthor         string
	initEmail          string
	initDescription    string
	initNonInteractive bool
	initNamespace      string
)

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.PersistentFlags().BoolVar(&initForce, "force", false, "Overwrite existing directory")
	initCmd.PersistentFlags().BoolVar(&initNoGit, "no-git", false, "Skip git initialization")
	initCmd.PersistentFlags().StringVar(&initAuthor, "author", "", "Author name for the project")
	initCmd.PersistentFlags().StringVar(&initEmail, "email", "", "Author email for the project")
	initCmd.PersistentFlags().StringVar(&initDescription, "description", "", "Description for the project")
	initCmd.PersistentFlags().BoolVar(&initNonInteractive, "non-interactive", false, "Run in non-interactive mode")
	initCmd.PersistentFlags().StringVar(&initNamespace, "namespace", "default", "Default namespace for project resources")
}

func runInit(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
	return nil
}

func runInitFramework(
	projectName, framework string,
	customizeProjectConfig func(*templates.ProjectConfig) error,
) error {

	// Validate project name
	if err := validateProjectName(projectName); err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}

	if !initNonInteractive {
		if initDescription == "" {
			initDescription = promptForDescription()
		}
		if initAuthor == "" {
			initAuthor = promptForAuthor()
		}
		if initEmail == "" {
			initEmail = promptForEmail()
		}
	}

	// Create project manifest
	projectManifest := manifest.GetDefault(projectName, framework, initDescription, initAuthor, initEmail, initNamespace)

	// Check if directory exists
	projectPath, err := filepath.Abs(projectName)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for project: %w", err)
	}

	// Create project configuration
	projectConfig := templates.ProjectConfig{
		ProjectName: projectManifest.Name,
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

	// Customize project config for the specific framework
	if customizeProjectConfig != nil {
		if err := customizeProjectConfig(&projectConfig); err != nil {
			return fmt.Errorf("failed to customize project config: %w", err)
		}
	}

	// Generate project files
	generator, err := frameworks.GetGenerator(framework)
	if err != nil {
		return err
	}
	if err := generator.GenerateProject(projectConfig); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}

	return manifest.NewManager(projectPath).Save(projectManifest)
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
