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
	updateFlagName        string
	updateFlagValue       string
	updateFlagType        string
	updateFlagDescription string
	updateFlagOverwrite   bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing SSM parameter",
	Long: `Update an existing SSM Parameter Store parameter.

When called with --name and --value flags, the parameter is updated directly.
When called without flags, an interactive TUI form is launched that pre-populates
the current parameter values for editing.

The --overwrite flag defaults to true for updates. Use --overwrite=false to fail
if the parameter already exists (useful for conditional writes).

Supported types: String, StringList, SecureString.

Examples:
  awse ssm update                                                     # interactive TUI form
  awse ssm update --name /app/config/db_host --value newhost.example.com
  awse ssm update --name /app/config/db_pass --value newpass --type SecureString
  awse ssm update --name /app/config/db_host --value newhost --overwrite=false`,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().StringVar(&updateFlagName, "name", "", "Parameter name (e.g. /app/config/db_host)")
	updateCmd.Flags().StringVar(&updateFlagValue, "value", "", "New parameter value")
	updateCmd.Flags().StringVar(&updateFlagType, "type", "", "Parameter type: String, StringList, or SecureString (defaults to existing type)")
	updateCmd.Flags().StringVar(&updateFlagDescription, "description", "", "Optional parameter description")
	updateCmd.Flags().BoolVar(&updateFlagOverwrite, "overwrite", true, "Overwrite the existing parameter (default true)")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Determine whether to use flags-based or TUI-based flow.
	nameSet := cmd.Flags().Changed("name")
	valueSet := cmd.Flags().Changed("value")

	if !nameSet && !valueSet {
		return runUpdateInteractive(cmd)
	}

	// Flags-based path: both --name and --value are required.
	if updateFlagName == "" {
		return fmt.Errorf("--name is required (or omit all flags for interactive mode)")
	}
	if updateFlagValue == "" {
		return fmt.Errorf("--value is required (or omit all flags for interactive mode)")
	}

	return executeUpdate(cmd, updateFlagName, updateFlagValue, updateFlagType, updateFlagDescription, updateFlagOverwrite)
}

// runUpdateInteractive launches the Bubble Tea TUI form pre-populated with the
// existing parameter's values for editing.
func runUpdateInteractive(cmd *cobra.Command) error {
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
	result, err := tui.RunSSMCreate("Update SSM Parameter", tui.SSMCreateResult{})
	if err != nil {
		return fmt.Errorf("parameter form error: %w", err)
	}
	if result == nil {
		return nil // user cancelled
	}

	// Wire profile/region into the SSM client for the API call.
	ctx := context.Background()
	client, err := ssmclient.NewClient(ctx, ssmclient.ClientOptions{
		Profile: profile,
		Region:  region,
	})
	if err != nil {
		return fmt.Errorf("failed to create SSM client: %w", err)
	}

	// Use UpdateParameter which enforces Overwrite=true for updates.
	version, err := client.UpdateParameter(ctx, ssmclient.UpdateParameterInput{
		Name:        result.Name,
		Value:       result.Value,
		Type:        result.Type,
		Description: result.Description,
	})
	if err != nil {
		return fmt.Errorf("failed to update parameter: %w", err)
	}

	// Success output to stderr (consistent with other awse commands).
	typeLabel := result.Type
	if typeLabel == "SecureString" {
		typeLabel = tui.Yellow.Render("SecureString")
	}

	fmt.Fprintf(os.Stderr, "%s Updated parameter %s (type: %s, version: %d)\n",
		tui.Green.Render("✔"),
		tui.Cyan.Render(result.Name),
		typeLabel,
		version,
	)

	return nil
}

// executeUpdate performs the SSM PutParameter call to update an existing parameter
// using explicit flag values. It first verifies the parameter exists and resolves
// the type if not explicitly provided.
func executeUpdate(cmd *cobra.Command, name, value, paramType, description string, overwrite bool) error {
	// Validate parameter name starts with /
	if !strings.HasPrefix(name, "/") {
		return fmt.Errorf("parameter name must start with '/': got %q", name)
	}

	// Validate value is not empty
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("parameter value must not be empty")
	}

	// Validate parameter type if explicitly provided
	if paramType != "" && !isValidType(paramType) {
		return fmt.Errorf("invalid parameter type %q: must be one of %s", paramType, strings.Join(validTypes, ", "))
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

	// If --type was not explicitly set, fetch the existing parameter to preserve its type.
	if !cmd.Flags().Changed("type") || paramType == "" {
		existing, err := client.GetParameter(ctx, name, false)
		if err != nil {
			return fmt.Errorf("parameter %q not found or inaccessible (use 'awse ssm create' for new parameters): %w", name, err)
		}
		paramType = existing.Type
	}

	var version int64
	if overwrite {
		// Use UpdateParameter which enforces Overwrite=true for the common update case.
		version, err = client.UpdateParameter(ctx, ssmclient.UpdateParameterInput{
			Name:        name,
			Value:       value,
			Type:        paramType,
			Description: description,
		})
	} else {
		// When --overwrite=false, use PutParameter directly for conditional writes.
		version, err = client.PutParameter(ctx, ssmclient.PutParameterInput{
			Name:        name,
			Value:       value,
			Type:        paramType,
			Description: description,
			Overwrite:   false,
		})
	}
	if err != nil {
		return fmt.Errorf("failed to update parameter: %w", err)
	}

	typeLabel := paramType
	if typeLabel == "SecureString" {
		typeLabel = tui.Yellow.Render("SecureString")
	}

	fmt.Fprintf(os.Stderr, "%s Updated parameter %s (type: %s, version: %d)\n",
		tui.Green.Render("✔"),
		tui.Cyan.Render(name),
		typeLabel,
		version,
	)

	return nil
}
