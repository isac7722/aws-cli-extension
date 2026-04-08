package ssm

import (
	"context"
	"fmt"
	"os"
	"strings"

	ssmclient "github.com/isac7722/aws-cli-extension/internal/ssm"
	"github.com/isac7722/aws-cli-extension/internal/tui"
	"github.com/spf13/cobra"
)

var (
	deleteFlagName string
	deleteFlagYes  bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an SSM parameter",
	Long: `Delete a single SSM Parameter Store parameter by name/path.

A confirmation prompt is shown before deletion unless --yes is used.

Examples:
  awse ssm delete --name /app/config/db_host
  awse ssm delete --name /app/config/old_key --yes
  awse ssm delete --name /app/config/old_key --profile production --region us-east-1`,
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().StringVar(&deleteFlagName, "name", "", "Parameter name/path to delete (e.g. /app/config/db_host)")
	deleteCmd.Flags().BoolVarP(&deleteFlagYes, "yes", "y", false, "Skip confirmation prompt")
	_ = deleteCmd.MarkFlagRequired("name")
}

func runDelete(cmd *cobra.Command, args []string) error {
	name := deleteFlagName

	// Validate parameter name starts with /
	if !strings.HasPrefix(name, "/") {
		return fmt.Errorf("parameter name must start with '/': got %q", name)
	}

	profile := flagProfile
	region := flagRegion

	ctx := context.Background()
	client, err := ssmclient.NewClient(ctx, ssmclient.ClientOptions{
		Profile: profile,
		Region:  region,
	})
	if err != nil {
		return fmt.Errorf("failed to create SSM client: %w", err)
	}

	// Confirmation prompt (skip with --yes)
	if !deleteFlagYes {
		confirmed, err := tui.RunDeleteConfirm(
			fmt.Sprintf("Delete parameter %q?", name),
			name,
		)
		if err != nil {
			return fmt.Errorf("confirm error: %w", err)
		}
		if !confirmed {
			fmt.Fprintln(os.Stderr, tui.Dim.Render("Cancelled."))
			return nil
		}
	}

	if err := client.DeleteParameter(ctx, name); err != nil {
		return fmt.Errorf("failed to delete parameter: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s Parameter %s deleted.\n",
		tui.Green.Render("✔"),
		tui.Cyan.Render(name),
	)

	return nil
}
