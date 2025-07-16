package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"kagent.dev/kmcp/pkg/tools"
)

var addToolCmd = &cobra.Command{
	Use:   "add-tool [tool-name]",
	Short: "Generate a new MCP tool with dynamic loading",
	Long: `Generate a new MCP tool that will be automatically loaded by the server.

This command creates a new tool file in src/tools/ with the same name as the tool.
The tool will be automatically discovered and loaded when the server starts.

Each tool is a Python file containing a function decorated with @mcp.tool().
The function should use the @mcp.tool() decorator from FastMCP.

Examples:
  kmcp add-tool weather
  kmcp add-tool database --description "Database operations tool"
  kmcp add-tool weather --force  # Overwrite existing tool
  
The generated tool file will include commented examples for common patterns:
- HTTP API calls
- Database operations
- File processing
- Configuration access

For API tools, configure environment variables in kmcp.yaml:
  tools:
    weather:
      api_key_env: "WEATHER_API_KEY"
      base_url: "https://api.weather.com"`,
	Args: cobra.ExactArgs(1),
	RunE: runAddTool,
}

var (
	addToolDescription string
	addToolForce       bool
	addToolInteractive bool
)

func init() {
	rootCmd.AddCommand(addToolCmd)

	addToolCmd.Flags().StringVarP(&addToolDescription, "description", "d", "", "Tool description")
	addToolCmd.Flags().BoolVarP(&addToolForce, "force", "f", false, "Overwrite existing tool file")
	addToolCmd.Flags().BoolVarP(&addToolInteractive, "interactive", "i", false, "Interactive tool creation")
}

func runAddTool(cmd *cobra.Command, args []string) error {
	toolName := args[0]

	// Validate tool name
	if err := validateToolName(toolName); err != nil {
		return fmt.Errorf("invalid tool name: %w", err)
	}

	// Check if we're in a valid KMCP project
	if !isKMCPProject() {
		return fmt.Errorf("not in a KMCP project directory. Run 'kmcp init' first")
	}

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
		return createToolInteractive(toolName, toolPath)
	}

	return createTool(toolName, toolPath)
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
	if !((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z') || name[0] == '_') {
		return false
	}

	// Remaining characters must be letters, digits, or underscores
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

func isKMCPProject() bool {
	return fileExists("kmcp.yaml") || fileExists("kmcp.yml")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func createToolInteractive(toolName, toolPath string) error {
	fmt.Printf("Creating tool '%s' interactively...\n", toolName)

	// Get tool description
	if addToolDescription == "" {
		fmt.Printf("Enter tool description (optional): ")
		var desc string
		fmt.Scanln(&desc)
		addToolDescription = desc
	}

	return generateTool(toolName, toolPath)
}

func createTool(toolName, toolPath string) error {
	if verbose {
		fmt.Printf("Creating tool: %s\n", toolName)
	}

	return generateTool(toolName, toolPath)
}

func generateTool(toolName, toolPath string) error {
	// Generate the tool file
	generator := tools.NewGenerator()

	config := map[string]interface{}{
		"description": addToolDescription,
	}

	if err := generator.GenerateToolFile(toolPath, toolName, config); err != nil {
		return fmt.Errorf("failed to generate tool file: %w", err)
	}

	fmt.Printf("✅ Successfully created tool: %s\n", toolName)
	fmt.Printf("📁 Generated file: %s\n", toolPath)
	
	// Regenerate __init__.py file
	toolsDir := filepath.Dir(toolPath)
	if err := generator.RegenerateToolsInit(toolsDir); err != nil {
		fmt.Printf("⚠️  Warning: Failed to regenerate __init__.py: %v\n", err)
	} else {
		fmt.Printf("🔄 Updated tools/__init__.py with new tool import\n")
	}

	fmt.Printf("📝 Edit the file to implement your tool logic\n")
	fmt.Printf("🚀 The tool will be automatically loaded when the server starts\n")

	if addToolDescription != "" {
		fmt.Printf("📋 Description: %s\n", addToolDescription)
	}

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Edit %s to implement your tool logic\n", toolPath)
	fmt.Printf("2. Configure any required environment variables in kmcp.yaml\n")
	fmt.Printf("3. Run 'uv run python src/main.py' to start the server\n")
	fmt.Printf("4. Run 'uv run pytest tests/' to test your tool\n")

	return nil
}
