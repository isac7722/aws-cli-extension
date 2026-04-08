package ssm

import (
	"context"
	"fmt"
	"os"
	"strings"

	ssmclient "github.com/isac7722/aws-cli-extension/internal/ssm"
	"github.com/isac7722/aws-cli-extension/internal/tui"
	uissm "github.com/isac7722/aws-cli-extension/internal/ui/ssm"
	"github.com/spf13/cobra"
)

var (
	putFlagName        string
	putFlagValue       string
	putFlagType        string
	putFlagDescription string
	putFlagOverwrite   bool
)

var putCmd = &cobra.Command{
	Use:   "put",
	Short: "Create or update an SSM parameter",
	Long: `Create or update an SSM Parameter Store parameter.

When called with --name and --value flags, the parameter is created directly.
When called without flags, an interactive TUI form is launched.

Supported types: String, StringList, SecureString.

Examples:
  awse ssm put                                                   # interactive TUI form
  awse ssm put --name /app/config/db_host --value mydb.example.com
  awse ssm put --name /app/config/db_pass --value s3cret --type SecureString
  awse ssm put --name /app/config/db_host --value newhost --overwrite`,
	RunE: runPut,
}

func init() {
	putCmd.Flags().StringVar(&putFlagName, "name", "", "Parameter name (e.g. /app/config/db_host)")
	putCmd.Flags().StringVar(&putFlagValue, "value", "", "Parameter value")
	putCmd.Flags().StringVar(&putFlagType, "type", "String", "Parameter type: String, StringList, or SecureString")
	putCmd.Flags().StringVar(&putFlagDescription, "description", "", "Optional parameter description")
	putCmd.Flags().BoolVar(&putFlagOverwrite, "overwrite", false, "Overwrite if the parameter already exists")
}

func runPut(cmd *cobra.Command, args []string) error {
	// Determine whether to use flags-based or TUI-based flow.
	// If neither --name nor --value was explicitly set, launch the interactive form.
	nameSet := cmd.Flags().Changed("name")
	valueSet := cmd.Flags().Changed("value")

	if !nameSet && !valueSet {
		return runPutInteractive(cmd)
	}

	// Flags-based path: both --name and --value are required.
	if putFlagName == "" {
		return fmt.Errorf("--name is required (or omit all flags for interactive mode)")
	}
	if putFlagValue == "" {
		return fmt.Errorf("--value is required (or omit all flags for interactive mode)")
	}

	return executePut(putFlagName, putFlagValue, putFlagType, putFlagDescription, putFlagOverwrite)
}

// runPutInteractive launches the Bubble Tea TUI form and on submission invokes
// the SSM PutParameter API call.
func runPutInteractive(cmd *cobra.Command) error {
	// Resolve profile — use parent ssm flag or prompt interactively.
	profile := flagProfile
	region := flagRegion

	if profile == "" {
		currentProfile := os.Getenv("AWS_PROFILE")
		chosen, err := uissm.RunProfileSelector(currentProfile)
		if err != nil {
			return fmt.Errorf("profile selector error: %w", err)
		}
		if chosen == nil {
			return nil // user cancelled
		}
		profile = chosen.Name
		if region == "" && chosen.Region != "" {
			region = chosen.Region
		}
	}

	if region == "" {
		currentRegion := os.Getenv("AWS_REGION")
		if currentRegion == "" {
			currentRegion = os.Getenv("AWS_DEFAULT_REGION")
		}
		chosenRegion, err := uissm.RunRegionSelector(currentRegion)
		if err != nil {
			return fmt.Errorf("region selector error: %w", err)
		}
		if chosenRegion == "" {
			return nil // user cancelled
		}
		region = chosenRegion
	}

	// Show the TUI form for parameter input.
	result, err := tui.RunSSMCreate("Create/Update SSM Parameter", tui.SSMCreateResult{})
	if err != nil {
		return fmt.Errorf("parameter form error: %w", err)
	}
	if result == nil {
		return nil // user cancelled
	}

	// Ask for overwrite confirmation — the form doesn't have an overwrite toggle,
	// so we default to overwrite=true when using the interactive path (the API
	// will return an error if the parameter exists and overwrite is false).
	overwrite := true

	// Wire profile/region into the SSM client for the API call.
	ctx := context.Background()
	client, err := ssmclient.NewClient(ctx, ssmclient.ClientOptions{
		Profile: profile,
		Region:  region,
	})
	if err != nil {
		return fmt.Errorf("failed to create SSM client: %w", err)
	}

	version, err := client.PutParameter(ctx, ssmclient.PutParameterInput{
		Name:        result.Name,
		Value:       result.Value,
		Type:        result.Type,
		Description: result.Description,
		Overwrite:   overwrite,
	})
	if err != nil {
		return fmt.Errorf("failed to put parameter: %w", err)
	}

	// Success output to stderr (consistent with other awse commands).
	typeLabel := result.Type
	if typeLabel == "SecureString" {
		typeLabel = tui.Yellow.Render("SecureString")
	}

	fmt.Fprintf(os.Stderr, "%s Put parameter %s (type: %s, version: %d)\n",
		tui.Green.Render("✔"),
		tui.Cyan.Render(result.Name),
		typeLabel,
		version,
	)

	return nil
}

// executePut performs the SSM PutParameter call using explicit flag values.
func executePut(name, value, paramType, description string, overwrite bool) error {
	// Validate parameter name starts with /
	if !strings.HasPrefix(name, "/") {
		return fmt.Errorf("parameter name must start with '/': got %q", name)
	}

	// Validate parameter type
	if !isValidType(paramType) {
		return fmt.Errorf("invalid parameter type %q: must be one of %s", paramType, strings.Join(validTypes, ", "))
	}

	// Validate value is not empty
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("parameter value must not be empty")
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

	version, err := client.PutParameter(ctx, ssmclient.PutParameterInput{
		Name:        name,
		Value:       value,
		Type:        paramType,
		Description: description,
		Overwrite:   overwrite,
	})
	if err != nil {
		return fmt.Errorf("failed to put parameter: %w", err)
	}

	typeLabel := paramType
	if typeLabel == "SecureString" {
		typeLabel = tui.Yellow.Render("SecureString")
	}

	fmt.Fprintf(os.Stderr, "%s Put parameter %s (type: %s, version: %d)\n",
		tui.Green.Render("✔"),
		tui.Cyan.Render(name),
		typeLabel,
		version,
	)

	return nil
}
