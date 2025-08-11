package commands

import "github.com/spf13/cobra"

var Verbose bool

var subcommands []*cobra.Command

func addRootSubCmd(cmd *cobra.Command) {
	subcommands = append(subcommands, cmd)
}

func GetSubCommands() []*cobra.Command {
	return subcommands
}
