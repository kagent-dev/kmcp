package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"kagent.dev/kmcp/pkg/build"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build MCP server",
	Long: `Build an MCP server from the current project.
	
This command will detect the project type and build the appropriate
MCP server package or Docker image.`,
	RunE: runBuild,
}

var (
	buildDocker   bool
	buildOutput   string
	buildTag      string
	buildPlatform string
)

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().BoolVar(&buildDocker, "docker", false, "Build Docker image")
	buildCmd.Flags().StringVarP(&buildOutput, "output", "o", "", "Output directory or image name")
	buildCmd.Flags().StringVarP(&buildTag, "tag", "t", "", "Docker image tag")
	buildCmd.Flags().StringVar(&buildPlatform, "platform", "", "Target platform (e.g., linux/amd64,linux/arm64)")
}

func runBuild(_ *cobra.Command, _ []string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if verbose {
		fmt.Printf("Building MCP server in: %s\n", cwd)
	}

	// Check if this is a valid MCP project
	if err := validateMCPProject(cwd); err != nil {
		return fmt.Errorf("invalid MCP project: %w", err)
	}

	// Create build options
	opts := build.Options{
		ProjectDir: cwd,
		Docker:     buildDocker,
		Output:     buildOutput,
		Tag:        buildTag,
		Platform:   buildPlatform,
		Verbose:    verbose,
	}

	// Execute build
	builder := build.New()
	return builder.Build(opts)
}

// validateMCPProject checks if the current directory contains a valid MCP project
func validateMCPProject(dir string) error {
	// Check for common MCP project indicators
	indicators := []string{
		"pyproject.toml",   // Python projects
		"package.json",     // Node.js projects
		"go.mod",           // Go projects
		".mcpbuilder.yaml", // Our project config
	}

	for _, indicator := range indicators {
		if _, err := os.Stat(filepath.Join(dir, indicator)); err == nil {
			if verbose {
				fmt.Printf("Detected project file: %s\n", indicator)
			}
			return nil
		}
	}

	return fmt.Errorf("no MCP project detected. Expected one of: %v", indicators)
}
