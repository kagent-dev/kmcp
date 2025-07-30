package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kagent-dev/kmcp/pkg/templates"

	"github.com/kagent-dev/kmcp/pkg/frameworks"
	"github.com/kagent-dev/kmcp/pkg/manifest"
	"github.com/spf13/cobra"
)

var addToolCmd = &cobra.Command{
	Use:   "add-tool [tool-name]",
	Short: "Add a new MCP tool to your project",
	Long: `Generate a new MCP tool that will be automatically loaded by the server.

This command creates a new tool file in src/tools/ with a generic template.
The tool will be automatically discovered and loaded when the server starts.

Each tool is a Python file containing a function decorated with @mcp.tool().
The function should use the @mcp.tool() decorator from FastMCP.

Examples:
  kmcp add-tool weather
  kmcp add-tool database --description "Database operations tool"
  kmcp add-tool weather --force  # Overwrite existing tool
`,
	Args: cobra.ExactArgs(1),
	RunE: runAddTool,
}

var (
	addToolDescription string
	addToolForce       bool
	addToolInteractive bool
	addToolDir         string
)

func init() {
	rootCmd.AddCommand(addToolCmd)

	addToolCmd.Flags().StringVarP(&addToolDescription, "description", "d", "", "Tool description")
	addToolCmd.Flags().BoolVarP(&addToolForce, "force", "f", false, "Overwrite existing tool file")
	addToolCmd.Flags().BoolVarP(&addToolInteractive, "interactive", "i", false, "Interactive tool creation")
	addToolCmd.Flags().StringVar(&addToolDir, "project-dir", "", "Project directory (default: current directory)")
}

func runAddTool(_ *cobra.Command, args []string) error {
	toolName := args[0]

	// Validate tool name
	if err := validateToolName(toolName); err != nil {
		return fmt.Errorf("invalid tool name: %w", err)
	}

	// Determine project directory
	projectDirectory := addToolDir
	if projectDirectory == "" {
		var err error
		projectDirectory, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	} else {
		// Convert relative path to absolute path
		if !filepath.IsAbs(projectDirectory) {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			projectDirectory = filepath.Join(cwd, projectDirectory)
		}
	}

	manifestManager := manifest.NewManager(projectDirectory)
	projectManifest, err := manifestManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load project manifest: %w", err)
	}
	framework := projectManifest.Framework

	// Check if tool already exists
	toolPath := filepath.Join("src", "tools", toolName+".py")
	toolExists := fileExists(toolPath)

	if verbose {
		fmt.Printf("Tool file path: %s\n", toolPath)
		fmt.Printf("Tool exists: %v\n", toolExists)
	}

	if toolExists && !addToolForce {
		return fmt.Errorf("tool '%s' already exists. Use --force to overwrite", toolName)
	}

	if addToolInteractive {
		return createToolInteractive(toolName, projectDirectory, framework)
	}

	return createTool(toolName, projectDirectory, framework)
}

func validateToolName(name string) error {
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	// Check for valid Python identifier
	if !isValidPythonIdentifier(name) {
		return fmt.Errorf("tool name must be a valid Python identifier")
	}

	// Check for reserved names
	reservedNames := []string{"server", "main", "core", "utils", "init", "test"}
	for _, reserved := range reservedNames {
		if strings.ToLower(name) == reserved {
			return fmt.Errorf("'%s' is a reserved name", name)
		}
	}

	return nil
}

func isValidPythonIdentifier(name string) bool {
	if len(name) == 0 {
		return false
	}

	// First character must be letter or underscore
	firstChar := name[0]
	if firstChar < 'a' || firstChar > 'z' {
		if firstChar < 'A' || firstChar > 'Z' {
			if firstChar != '_' {
				return false
			}
		}
	}

	// Remaining characters must be letters, digits, or underscores
	for i := 1; i < len(name); i++ {
		c := name[i]
		if c < 'a' || c > 'z' {
			if c < 'A' || c > 'Z' {
				if c < '0' || c > '9' {
					if c != '_' {
						return false
					}
				}
			}
		}
	}

	return true
}

func createToolInteractive(toolName, projectRoot, framework string) error {
	fmt.Printf("Creating tool '%s' interactively...\n", toolName)

	// Get tool description
	if addToolDescription == "" {
		fmt.Printf("Enter tool description (optional): ")
		var desc string
		_, err := fmt.Scanln(&desc)
		if err != nil {
			return fmt.Errorf("failed to read description: %w", err)
		}
		addToolDescription = desc
	}

	return generateTool(toolName, projectRoot, framework)
}

func createTool(toolName, projectRoot, framework string) error {
	if verbose {
		fmt.Printf("Creating tool: %s\n", toolName)
	}

	return generateTool(toolName, projectRoot, framework)
}

func generateTool(toolName, projectRoot, framework string) error {
	generator, err := frameworks.GetGenerator(framework)
	if err != nil {
		return err
	}

	config := templates.ToolConfig{
		ToolName:    toolName,
		Description: addToolDescription,
	}

	if err := generator.GenerateTool(projectRoot, config); err != nil {
		return fmt.Errorf("failed to generate tool file: %w", err)
	}

	return nil
}
