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
	batchDeleteFlagNames []string
	batchDeleteFlagYes   bool
)

var batchDeleteCmd = &cobra.Command{
	Use:   "batch-delete",
	Short: "Delete multiple SSM parameters at once",
	Long: `Delete multiple SSM Parameter Store parameters in a single operation.

AWS supports deleting up to 10 parameters per API call. When more than 10
parameter names are provided, they are automatically chunked into multiple
API calls.

A confirmation prompt is shown before deletion unless --yes is used.

Examples:
  awse ssm batch-delete --name /app/config/old1 --name /app/config/old2
  awse ssm batch-delete --name /app/config/key1 --name /app/config/key2 --yes
  awse ssm batch-delete --name /app/config/key1 --name /app/config/key2 --profile production --region us-east-1`,
	RunE: runBatchDelete,
}

func init() {
	batchDeleteCmd.Flags().StringArrayVar(&batchDeleteFlagNames, "name", nil, "Parameter name/path to delete (can be specified multiple times)")
	batchDeleteCmd.Flags().BoolVarP(&batchDeleteFlagYes, "yes", "y", false, "Skip confirmation prompt")
	_ = batchDeleteCmd.MarkFlagRequired("name")
}

func runBatchDelete(cmd *cobra.Command, args []string) error {
	names := batchDeleteFlagNames

	// Validate all parameter names start with /
	for _, name := range names {
		if !strings.HasPrefix(name, "/") {
			return fmt.Errorf("parameter name must start with '/': got %q", name)
		}
	}

	// Deduplicate names
	names = dedup(names)

	if len(names) == 0 {
		return fmt.Errorf("at least one parameter name is required")
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
	if !batchDeleteFlagYes {
		confirmed, err := tui.RunBatchDeleteConfirm(
			fmt.Sprintf("Delete %d parameter(s)?", len(names)),
			names,
		)
		if err != nil {
			return fmt.Errorf("confirm error: %w", err)
		}
		if !confirmed {
			fmt.Fprintln(os.Stderr, tui.Dim.Render("Cancelled."))
			return nil
		}
	}

	result, err := client.DeleteParameters(ctx, names)
	if err != nil {
		return fmt.Errorf("failed to delete parameters: %w", err)
	}

	// Report results
	if len(result.DeletedParameters) > 0 {
		fmt.Fprintf(os.Stderr, "%s Deleted %d parameter(s):\n",
			tui.Green.Render("✔"),
			len(result.DeletedParameters),
		)
		for _, name := range result.DeletedParameters {
			fmt.Fprintf(os.Stderr, "    %s\n", tui.Cyan.Render(name))
		}
	}

	if len(result.InvalidParameters) > 0 {
		fmt.Fprintf(os.Stderr, "%s %d parameter(s) not found or invalid:\n",
			tui.Yellow.Render("⚠"),
			len(result.InvalidParameters),
		)
		for _, name := range result.InvalidParameters {
			fmt.Fprintf(os.Stderr, "    %s\n", tui.Dim.Render(name))
		}
	}

	return nil
}

// dedup removes duplicate strings while preserving order.
func dedup(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}
