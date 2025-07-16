package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"kagent.dev/kmcp/pkg/tools"
)

var regenerateInitCmd = &cobra.Command{
	Use:   "regenerate-init",
	Short: "Regenerate the tools/__init__.py file",
	Long: `Regenerate the tools/__init__.py file based on the current tools in the directory.

This command scans the src/tools/ directory for Python files and regenerates the __init__.py
file with the appropriate imports and __all__ list. This is useful when:

- You've manually added tool files
- The __init__.py file is out of sync  
- You need to refresh the tool imports

The command will:
1. Scan src/tools/ for all .py files (excluding __init__.py)
2. Generate import statements for each tool
3. Update the __all__ list with all discovered tools
4. Overwrite the existing __init__.py file

Examples:
  kmcp regenerate-init                    # Regenerate in current directory
  kmcp regenerate-init --tools-dir custom-tools  # Use custom tools directory`,
	RunE: runRegenerateInit,
}

var regenerateInitToolsDir string

func init() {
	rootCmd.AddCommand(regenerateInitCmd)
	regenerateInitCmd.Flags().StringVar(&regenerateInitToolsDir, "tools-dir", "src/tools", "Tools directory to regenerate __init__.py for")
}

func runRegenerateInit(cmd *cobra.Command, args []string) error {
	// Check if we're in a valid KMCP project
	if !isKMCPProject() {
		return fmt.Errorf("not in a KMCP project directory. Run 'kmcp init' first")
	}

	// Check if tools directory exists
	if !fileExists(regenerateInitToolsDir) {
		return fmt.Errorf("tools directory '%s' does not exist", regenerateInitToolsDir)
	}

	// Get absolute path
	absToolsDir, err := filepath.Abs(regenerateInitToolsDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if verbose {
		fmt.Printf("Regenerating __init__.py in: %s\n", absToolsDir)
	}

	// Create generator and regenerate
	generator := tools.NewGenerator()
	if err := generator.RegenerateToolsInit(absToolsDir); err != nil {
		return fmt.Errorf("failed to regenerate __init__.py: %w", err)
	}

	fmt.Printf("✅ Successfully regenerated %s/__init__.py\n", regenerateInitToolsDir)

	return nil
} 