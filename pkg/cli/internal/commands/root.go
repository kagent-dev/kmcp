package commands

import (
	"github.com/spf13/cobra"
)

// Version will be set by ldflags during build
var Version = ""

var rootCmd = &cobra.Command{
	Use:   "kmcp",
	Short: "KMCP - Kubernetes Model Context Protocol CLI",
	Long: `KMCP is a CLI tool for building and managing Model Context Protocol (MCP) servers.
	
It provides a unified development experience for creating, building, and deploying
MCP servers locally and to Kubernetes clusters.`,
	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
}

var Verbose bool
