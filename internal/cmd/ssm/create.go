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

// Valid SSM parameter types.
var validTypes = []string{"String", "StringList", "SecureString"}

var (
	createFlagName        string
	createFlagValue       string
	createFlagType        string
	createFlagDescription string
	createFlagOverwrite   bool
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new SSM parameter",
	Long: `Create a new SSM Parameter Store parameter.

Requires --name and --value flags. The --type flag defaults to "String".
Supported types: String, StringList, SecureString.

Examples:
  awse ssm create --name /app/config/db_host --value mydb.example.com
  awse ssm create --name /app/config/db_pass --value s3cret --type SecureString
  awse ssm create --name /app/config/regions --value "us-east-1,us-west-2" --type StringList
  awse ssm create --name /app/config/db_host --value newhost --description "Database host" --overwrite`,
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringVar(&createFlagName, "name", "", "Parameter name (required, e.g. /app/config/db_host)")
	createCmd.Flags().StringVar(&createFlagValue, "value", "", "Parameter value (required)")
	createCmd.Flags().StringVar(&createFlagType, "type", "String", "Parameter type: String, StringList, or SecureString")
	createCmd.Flags().StringVar(&createFlagDescription, "description", "", "Optional parameter description")
	createCmd.Flags().BoolVar(&createFlagOverwrite, "overwrite", false, "Overwrite if the parameter already exists")

	_ = createCmd.MarkFlagRequired("name")
	_ = createCmd.MarkFlagRequired("value")
}

func runCreate(cmd *cobra.Command, args []string) error {
	// Validate parameter name starts with /
	if !strings.HasPrefix(createFlagName, "/") {
		return fmt.Errorf("parameter name must start with '/': got %q", createFlagName)
	}

	// Validate parameter type
	if !isValidType(createFlagType) {
		return fmt.Errorf("invalid parameter type %q: must be one of %s", createFlagType, strings.Join(validTypes, ", "))
	}

	// Validate value is not empty
	if strings.TrimSpace(createFlagValue) == "" {
		return fmt.Errorf("parameter value must not be empty")
	}

	// Build client options from parent ssm command flags
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

	version, err := client.PutParameter(ctx, ssmclient.PutParameterInput{
		Name:        createFlagName,
		Value:       createFlagValue,
		Type:        createFlagType,
		Description: createFlagDescription,
		Overwrite:   createFlagOverwrite,
	})
	if err != nil {
		return fmt.Errorf("failed to create parameter: %w", err)
	}

	// Success output to stderr (consistent with other awse commands)
	typeLabel := createFlagType
	if typeLabel == "SecureString" {
		typeLabel = tui.Yellow.Render("SecureString")
	}

	fmt.Fprintf(os.Stderr, "%s Created parameter %s (type: %s, version: %d)\n",
		tui.Green.Render("✔"),
		tui.Cyan.Render(createFlagName),
		typeLabel,
		version,
	)

	return nil
}

func isValidType(t string) bool {
	for _, v := range validTypes {
		if v == t {
			return true
		}
	}
	return false
}
