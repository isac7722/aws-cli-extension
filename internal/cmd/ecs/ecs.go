package ecs

import (
	"github.com/spf13/cobra"
)

var (
	flagProfile string
	flagRegion  string
)

// Cmd is the ecs parent command.
var Cmd = &cobra.Command{
	Use:   "ecs",
	Short: "Manage ECS services and deployments",
	Long:  "Interactive TUI for managing AWS ECS clusters, services, and deployments.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	Cmd.PersistentFlags().StringVar(&flagProfile, "profile", "", "AWS profile to use (overrides AWS_PROFILE)")
	Cmd.PersistentFlags().StringVar(&flagRegion, "region", "", "AWS region to use (overrides AWS_REGION)")

	Cmd.AddCommand(deployCmd)
}
