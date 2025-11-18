package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/kagent-dev/kmcp/pkg/cli/internal/commands"
	"github.com/kagent-dev/kmcp/pkg/cli/internal/themes"
	"github.com/kagent-dev/kmcp/pkg/internal/version"
	"github.com/spf13/cobra"
)

// Root returns the root command for the kmcp CLI
func Root() *cobra.Command {
	return rootCmd(version.GetVersion())
}

func rootCmd(version string) *cobra.Command {

	rootCmd := &cobra.Command{
		Use:   "kmcp",
		Short: "KMCP - Kubernetes Model Context Protocol CLI",
		Long: fmt.Sprintf(`%s
		
KMCP is a CLI tool for building and managing Model Context Protocol (MCP) servers.
		
It provides a unified development experience for creating, building, and deploying
MCP servers locally and to Kubernetes clusters.`, themes.ColoredKmcpLogo()),
		Version: version,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&commands.Verbose, "verbose", "v", false, "verbose output")

	cobra.AddTemplateFunc("sectionHeader", sectionHeader)

	rootCmd.SetHelpTemplate(helpTemplate)

	rootCmd.AddCommand(commands.GetSubCommands()...)

	return rootCmd
}

const helpTemplate = `{{with (or .Long .Short)}}{{ . | trimTrailingWhitespaces}}{{end}}

{{sectionHeader "Usage:"}}
  {{.UseLine}}{{if .HasAvailableSubCommands}}

{{sectionHeader "Available Commands:"}}:{{range .Commands}}{{if (not .Hidden)}}
  {{rpad .Name .NamePadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasExample}}

{{sectionHeader "Examples:"}}
{{.Example | trimTrailingWhitespaces}}{{end}}{{if .HasLocalFlags}}

{{sectionHeader "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasInheritedFlags}}

{{sectionHeader "GlobalFlags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

`

// sectionHeader returns a styled section header, using the kmcp color
func sectionHeader(s string) string {
	return themes.ColorPrimary().Add(color.Bold).Sprint(s)
}
