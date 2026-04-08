package cmd

import (
	"github.com/isac7722/aws-cli-extension/internal/cmd/ecs"
	"github.com/isac7722/aws-cli-extension/internal/cmd/ssm"
	"github.com/isac7722/aws-cli-extension/internal/cmd/user"
	"github.com/spf13/cobra"
)

var versionStr = "dev"

// SetVersionInfo sets version info from ldflags.
func SetVersionInfo(v, _, _ string) {
	versionStr = v
}

var rootCmd = &cobra.Command{
	Use:   "awse",
	Short: "AWS Extension — interactive AWS profile and parameter management",
	Long: `awse is a lightweight CLI extending AWS workflows with interactive profile management,
SSM Parameter Store browsing, and AWS CLI installation guidance.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(user.Cmd)
	rootCmd.AddCommand(ssm.Cmd)
	rootCmd.AddCommand(ecs.Cmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
