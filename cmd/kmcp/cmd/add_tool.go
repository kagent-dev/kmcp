package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"kagent.dev/kmcp/pkg/manifest"
	"kagent.dev/kmcp/pkg/tools"
)

var addToolCmd = &cobra.Command{
	Use:   "add-tool [tool-name]",
	Short: "Generate and register a new MCP tool",
	Long: `Generate and register a new MCP tool in one step.

This command will:
1. Generate the tool code from a template (if it doesn't exist)
2. Register it with the MCP server
3. Update the project manifest
4. Restart the dev server if running

For security, API keys are accessed via environment variables, not stored in config files.

Examples:
  kmcp add-tool weather --type api-client
  kmcp add-tool weather --type api-client --url https://api.weather.com
  kmcp add-tool weather --interactive
  kmcp add-tool weather (auto-discover existing file)
  kmcp add-tool weather --type basic -f (overwrite existing)
  
For API client tools, set your auth key in environment:
  export WEATHER_API_KEY=your_api_key_here
  # or add to .env.local: WEATHER_API_KEY=your_api_key_here`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runAddTool,
}

var (
	addToolType        string
	addToolURL         string
	addToolDescription string
	addToolForce       bool
	addToolInteractive bool
	addToolListTypes   bool
	addToolDiscover    bool
)

func init() {
	rootCmd.AddCommand(addToolCmd)

	addToolCmd.Flags().StringVarP(&addToolType, "type", "t", "", "Tool type (basic, api-client, database, filesystem, multi-step)")
	addToolCmd.Flags().StringVar(&addToolURL, "url", "", "API URL for api-client type")
	addToolCmd.Flags().StringVarP(&addToolDescription, "description", "d", "", "Tool description")
	addToolCmd.Flags().BoolVarP(&addToolForce, "force", "f", false, "Overwrite existing tool file")
	addToolCmd.Flags().BoolVarP(&addToolInteractive, "interactive", "i", false, "Interactive tool creation")
	addToolCmd.Flags().BoolVar(&addToolListTypes, "list-types", false, "List available tool types")
	addToolCmd.Flags().BoolVar(&addToolDiscover, "discover", false, "Auto-discover existing tool file")
}

func runAddTool(cmd *cobra.Command, args []string) error {
	// Handle special flags
	if addToolListTypes {
		return listToolTypes()
	}

	if len(args) == 0 {
		return fmt.Errorf("tool name is required")
	}

	toolName := args[0]

	// Validate tool name
	if err := validateToolName(toolName); err != nil {
		return fmt.Errorf("invalid tool name: %w", err)
	}

	// Check if we're in a valid KMCP project
	if !isKMCPProject() {
		return fmt.Errorf("not in a KMCP project directory. Run 'kmcp init' first")
	}

	// Load project manifest
	manifestManager := manifest.NewManager(".")
	projectManifest, err := manifestManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load project manifest: %w", err)
	}

	// Check if tool already exists
	toolPath := filepath.Join("src", "tools", toolName+".py")
	toolExists := fileExists(toolPath)

	if verbose {
		fmt.Printf("Tool file path: %s\n", toolPath)
		fmt.Printf("Tool exists: %v\n", toolExists)
	}

	// Handle different scenarios
	if toolExists && !addToolForce {
		if addToolDiscover || addToolType == "" {
			// Auto-discover and register existing tool
			return discoverAndRegisterTool(toolName, toolPath, projectManifest, manifestManager)
		} else {
			return fmt.Errorf("tool '%s' already exists. Use -f to overwrite or omit --type to register existing tool", toolName)
		}
	}

	// Generate new tool or overwrite existing
	if addToolType == "" && !toolExists {
		return fmt.Errorf("tool type is required for new tools. Use --type or --interactive")
	}

	if addToolInteractive {
		return createToolInteractive(toolName, toolPath, projectManifest, manifestManager)
	}

	return createTool(toolName, toolPath, addToolType, projectManifest, manifestManager)
}

func listToolTypes() error {
	fmt.Println("Available tool types:")
	fmt.Println("  basic     - Minimal tool structure with basic functionality")
	fmt.Println("  http      - Tool with HTTP client capabilities")
	fmt.Println("  data      - Tool for data processing and manipulation")
	fmt.Println("  workflow  - Tool for multi-step operations and workflows")
	fmt.Println("\nUsage:")
	fmt.Println("  kmcp add-tool mytool --type basic")
	fmt.Println("  kmcp add-tool weather --type http")
	fmt.Println("  kmcp add-tool processor --type data")
	fmt.Println("  kmcp add-tool pipeline --type workflow")
	fmt.Println("\nFor HTTP tools, you can specify configuration:")
	fmt.Println("  kmcp add-tool weather --type http --description 'Weather API client'")
	return nil
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
	reservedNames := []string{"server", "registry", "main", "core", "config"}
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

func discoverAndRegisterTool(toolName, toolPath string, projectManifest *manifest.ProjectManifest, manifestManager *manifest.Manager) error {
	if verbose {
		fmt.Printf("Discovering tool: %s\n", toolName)
	}

	// Parse the existing tool file
	toolDiscovery := tools.NewDiscovery(".")
	toolInfo, err := toolDiscovery.AnalyzeToolFile(toolPath)
	if err != nil {
		return fmt.Errorf("failed to analyze tool file: %w", err)
	}

	// Register the discovered tool
	if err := registerTool(toolName, toolInfo, projectManifest, manifestManager); err != nil {
		return fmt.Errorf("failed to register tool: %w", err)
	}

	fmt.Printf("✅ Successfully registered existing tool: %s\n", toolName)
	if len(toolInfo.Methods) > 0 {
		fmt.Printf("📋 Discovered methods:\n")
		for _, method := range toolInfo.Methods {
			fmt.Printf("  - %s: %s\n", method.Name, method.Description)
		}
	}

	return nil
}

func createToolInteractive(toolName, toolPath string, projectManifest *manifest.ProjectManifest, manifestManager *manifest.Manager) error {
	fmt.Printf("Creating tool '%s' interactively...\n", toolName)

	// Get tool type
	toolType, err := promptForToolType()
	if err != nil {
		return fmt.Errorf("failed to get tool type: %w", err)
	}

	// Get type-specific configuration
	config, err := promptForToolConfig(toolType)
	if err != nil {
		return fmt.Errorf("failed to get tool configuration: %w", err)
	}

	return generateAndRegisterTool(toolName, toolPath, toolType, config, projectManifest, manifestManager)
}

func createTool(toolName, toolPath, toolType string, projectManifest *manifest.ProjectManifest, manifestManager *manifest.Manager) error {
	if verbose {
		fmt.Printf("Creating tool: %s (type: %s)\n", toolName, toolType)
	}

	// Build configuration from command line flags
	config := map[string]interface{}{
		"description": addToolDescription,
	}

	// Add type-specific configuration
	switch toolType {
	case "http":
		if addToolURL != "" {
			config["base_url"] = addToolURL
		}
	}

	return generateAndRegisterTool(toolName, toolPath, toolType, config, projectManifest, manifestManager)
}

func generateAndRegisterTool(toolName, toolPath, toolType string, config map[string]interface{}, projectManifest *manifest.ProjectManifest, manifestManager *manifest.Manager) error {
	// Generate the tool file
	generator := tools.NewGenerator()
	if err := generator.GenerateToolFile(toolPath, toolName, toolType, config); err != nil {
		return fmt.Errorf("failed to generate tool file: %w", err)
	}

	// Analyze the generated tool
	toolDiscovery := tools.NewDiscovery(".")
	toolInfo, err := toolDiscovery.AnalyzeToolFile(toolPath)
	if err != nil {
		return fmt.Errorf("failed to analyze generated tool: %w", err)
	}

	// Register the tool
	if err := registerTool(toolName, toolInfo, projectManifest, manifestManager); err != nil {
		return fmt.Errorf("failed to register tool: %w", err)
	}

	fmt.Printf("✅ Successfully created and registered tool: %s\n", toolName)
	fmt.Printf("📁 Generated file: %s\n", toolPath)
	if len(toolInfo.Methods) > 0 {
		fmt.Printf("📋 Generated methods:\n")
		for _, method := range toolInfo.Methods {
			fmt.Printf("  - %s: %s\n", method.Name, method.Description)
		}
	}

	// Restart dev server if running
	if err := restartDevServer(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to restart dev server: %v\n", err)
	}

	return nil
}

func registerTool(toolName string, toolInfo *tools.ToolInfo, projectManifest *manifest.ProjectManifest, manifestManager *manifest.Manager) error {
	// Update project manifest with tool information
	if projectManifest.Tools == nil {
		projectManifest.Tools = make(map[string]manifest.ToolConfig)
	}

	// Register each method as a separate tool
	for _, method := range toolInfo.Methods {
		toolConfig := manifest.ToolConfig{
			Name:        method.Name,
			Description: method.Description,
			Handler:     fmt.Sprintf("%s.%s", toolInfo.ClassName, method.Name),
			Config: map[string]interface{}{
				"enabled":         true,
				"auto_discovered": true,
				"file":            toolInfo.FilePath,
				"class":           toolInfo.ClassName,
				"method":          method.Name,
			},
		}

		// Add parameter information if available
		if len(method.Parameters) > 0 {
			toolConfig.Config["parameters"] = method.Parameters
		}

		projectManifest.Tools[method.Name] = toolConfig
	}

	// Save updated manifest
	if err := manifestManager.Save(projectManifest); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	return nil
}

func promptForToolType() (string, error) {
	fmt.Println("\nSelect tool type:")
	fmt.Println("1. Basic - Minimal tool structure with basic functionality")
	fmt.Println("2. HTTP - Tool with HTTP client capabilities")
	fmt.Println("3. Data - Tool for data processing and manipulation")
	fmt.Println("4. Workflow - Tool for multi-step operations and workflows")
	fmt.Print("Enter choice [1-4]: ")

	var choice string
	if _, err := fmt.Scanln(&choice); err != nil {
		return "", err
	}

	switch strings.TrimSpace(choice) {
	case "1", "":
		return "basic", nil
	case "2":
		return "http", nil
	case "3":
		return "data", nil
	case "4":
		return "workflow", nil
	default:
		return "basic", nil
	}
}

func promptForToolConfig(toolType string) (map[string]interface{}, error) {
	config := make(map[string]interface{})

	switch toolType {
	case "http":
		fmt.Print("Enter base URL (optional): ")
		var baseURL string
		if _, err := fmt.Scanln(&baseURL); err == nil && baseURL != "" {
			config["base_url"] = baseURL
		}

		fmt.Print("Enter timeout in seconds [30]: ")
		var timeout string
		if _, err := fmt.Scanln(&timeout); err == nil && timeout != "" {
			config["timeout"] = timeout
		}

	case "workflow":
		fmt.Print("Enter maximum steps [10]: ")
		var maxSteps string
		if _, err := fmt.Scanln(&maxSteps); err == nil && maxSteps != "" {
			config["max_steps"] = maxSteps
		}
	}

	return config, nil
}

func restartDevServer() error {
	// TODO: Implement dev server restart logic
	// This will be implemented when we add the dev command
	if verbose {
		fmt.Println("Dev server restart not yet implemented")
	}
	return nil
}
