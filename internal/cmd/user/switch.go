package user

import (
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:     "switch",
	Short:   "Interactively switch the active AWS profile",
	Aliases: []string{"sw"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInteractiveSwitch(cmd)
	},
}
