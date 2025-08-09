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

	root.AddCommand(commands.GetSubCommands()...)

	return root
}

func RootWithVersion(version string) *cobra.Command {
	return rootCmd(version)
}

func Root() *cobra.Command {
	return rootCmd(version.GetVersion())
}
