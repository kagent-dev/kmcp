package cli

import (
	"github.com/kagent-dev/kmcp/pkg/cli/internal/commands"
	"github.com/kagent-dev/kmcp/pkg/internal/version"
	"github.com/spf13/cobra"
)

func rootCmd(version string) *cobra.Command {

	root := &cobra.Command{
		Use:   "kmcp",
		Short: "KMCP - Kubernetes Model Context Protocol CLI",
		Long: `KMCP is a CLI tool for building and managing Model Context Protocol (MCP) servers.
		
	It provides a unified development experience for creating, building, and deploying
	MCP servers locally and to Kubernetes clusters.`,
		Version: version,
	}

	root.PersistentFlags().BoolVarP(&commands.Verbose, "verbose", "v", false, "verbose output")

	return root
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd(version.GetVersion()).Execute()
}

func ExecuteWithVersion(version string) error {
	return rootCmd(version).Execute()
}
