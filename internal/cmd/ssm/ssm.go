package ssm

import (
	"fmt"
	"os"

	uissm "github.com/isac7722/aws-cli-extension/internal/ui/ssm"
	"github.com/spf13/cobra"
)

var (
	flagPrefix  string
	flagProfile string
	flagRegion  string
)

// Cmd is the ssm parent command.
var Cmd = &cobra.Command{
	Use:   "ssm",
	Short: "Browse and manage SSM Parameter Store parameters",
	Long:  "Interactive TUI for browsing, viewing, and managing AWS SSM Parameter Store parameters.",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile := flagProfile
		region := flagRegion

		// If --profile was not provided, show the interactive profile selector.
		if profile == "" {
			currentProfile := os.Getenv("AWS_PROFILE")
			chosen, err := uissm.RunProfileSelector(currentProfile)
			if err != nil {
				return fmt.Errorf("profile selector error: %w", err)
			}
			if chosen == nil {
				// User cancelled the selector.
				return nil
			}
			profile = chosen.Name

			// Use the profile's region as default for the region selector.
			if region == "" && chosen.Region != "" {
				region = chosen.Region
			}
		}

		// If --region was not provided (and not derived from profile), show the region selector.
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
				// User cancelled the selector.
				return nil
			}
			region = chosenRegion
		}

		_, selectedValue, err := uissm.RunBrowser(uissm.BrowserOptions{
			Prefix:  flagPrefix,
			Profile: profile,
			Region:  region,
		})
		if err != nil {
			return fmt.Errorf("SSM browser error: %w", err)
		}

		// If the user selected a parameter value, output it to stdout.
		if selectedValue != "" {
			fmt.Println(selectedValue)
		}
		return nil
	},
}

func init() {
	Cmd.PersistentFlags().StringVar(&flagProfile, "profile", "", "AWS profile to use (overrides AWS_PROFILE)")
	Cmd.PersistentFlags().StringVar(&flagRegion, "region", "", "AWS region to use (overrides AWS_REGION)")
	Cmd.Flags().StringVarP(&flagPrefix, "prefix", "p", "/", "Starting parameter path prefix")

	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(putCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(batchDeleteCmd)
}
