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
	getFlagName    string
	getFlagDecrypt bool
	getFlagJSON    bool
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an SSM parameter value",
	Long: `Fetch a single SSM Parameter Store parameter by name/path.

The parameter value is printed to stdout, making it easy to capture in scripts.
Metadata (name, type, version, ARN) is printed to stderr for visibility.

By default, SecureString parameters are NOT decrypted. Use --decrypt to
request KMS decryption.

Examples:
  awse ssm get --name /app/config/db_host
  awse ssm get --name /app/config/db_pass --decrypt
  awse ssm get --name /app/config/db_host --profile production --region us-east-1`,
	RunE: runGet,
}

func init() {
	getCmd.Flags().StringVar(&getFlagName, "name", "", "Parameter name/path (e.g. /app/config/db_host)")
	getCmd.Flags().BoolVar(&getFlagDecrypt, "decrypt", false, "Decrypt SecureString values via KMS")
	_ = getCmd.MarkFlagRequired("name")
}

func runGet(cmd *cobra.Command, args []string) error {
	name := getFlagName

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

	result, err := client.GetParameter(ctx, name, getFlagDecrypt)
	if err != nil {
		return fmt.Errorf("failed to get parameter: %w", err)
	}

	// Print metadata to stderr for visibility.
	typeLabel := result.Type
	if typeLabel == "SecureString" {
		if getFlagDecrypt {
			typeLabel = tui.Yellow.Render("SecureString") + tui.Dim.Render(" (decrypted)")
		} else {
			typeLabel = tui.Yellow.Render("SecureString") + tui.Dim.Render(" (encrypted)")
		}
	}

	fmt.Fprintf(os.Stderr, "%s %s\n",
		tui.Green.Render("✔"),
		tui.Cyan.Render(result.Name),
	)
	fmt.Fprintf(os.Stderr, "  Type:    %s\n", typeLabel)
	fmt.Fprintf(os.Stderr, "  Version: %d\n", result.Version)
	if result.ARN != "" {
		fmt.Fprintf(os.Stderr, "  ARN:     %s\n", tui.Dim.Render(result.ARN))
	}
	if !result.LastModified.IsZero() {
		fmt.Fprintf(os.Stderr, "  Modified: %s\n", tui.Dim.Render(result.LastModified.Format("2006-01-02 15:04:05 MST")))
	}

	// Print value to stdout so it can be captured by scripts.
	fmt.Print(result.Value)

	return nil
}
