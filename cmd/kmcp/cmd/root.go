package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kmcp",
	Short: "KMCP - Kubernetes Model Context Protocol CLI",
	Long: `KMCP is a CLI tool for building and managing Model Context Protocol (MCP) servers.
	
It provides a unified development experience for creating, building, and deploying
MCP servers locally and to Kubernetes clusters.`,
	Version: "0.1.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

var verbose bool
