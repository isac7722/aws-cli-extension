package user

import (
	"fmt"
	"os"
	"strings"

	"github.com/isac7722/aws-cli-extension/internal/config"
	"github.com/isac7722/aws-cli-extension/internal/tui"
	"github.com/spf13/cobra"
)

// Common AWS regions for validation hints.
var commonRegions = []string{
	"us-east-1", "us-east-2", "us-west-1", "us-west-2",
	"eu-west-1", "eu-west-2", "eu-central-1",
	"ap-northeast-1", "ap-northeast-2", "ap-southeast-1", "ap-southeast-2",
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new AWS profile interactively",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Profile name
		profileName, ok, err := tui.RunPrompt("Profile name:", "e.g., work, personal, default")
		if err != nil {
			return err
		}
		if !ok || strings.TrimSpace(profileName) == "" {
			return nil
		}
		profileName = strings.TrimSpace(profileName)

		// Check for duplicate profile
		cfg, err := config.LoadProfiles()
		if err != nil {
			cfg = &config.AWSConfig{}
		}
		if _, exists := cfg.Get(profileName); exists {
			return fmt.Errorf("profile %q already exists. Use 'awse user edit %s' to modify it", profileName, profileName)
		}

		// AWS Access Key ID
		accessKey, ok, err := tui.RunPrompt("AWS Access Key ID:", "e.g., AKIAIOSFODNN7EXAMPLE")
		if err != nil {
			return err
		}
		if !ok || strings.TrimSpace(accessKey) == "" {
			return nil
		}
		accessKey = strings.TrimSpace(accessKey)

		// Validate access key format (starts with AKIA and is 20 chars)
		if !strings.HasPrefix(accessKey, "AKIA") || len(accessKey) != 20 {
			fmt.Fprintf(os.Stderr, "%s Access key format looks unusual (expected AKIA... with 20 characters)\n", tui.Yellow.Render("!"))
		}

		// AWS Secret Access Key (masked input)
		secretKey, ok, err := tui.RunSecretPrompt("AWS Secret Access Key:", "your secret access key")
		if err != nil {
			return err
		}
		if !ok || strings.TrimSpace(secretKey) == "" {
			return nil
		}
		secretKey = strings.TrimSpace(secretKey)

		// Default region
		region, _, err := tui.RunPrompt("Default region (optional):", "e.g., us-east-1, ap-northeast-2")
		if err != nil {
			return err
		}
		region = strings.TrimSpace(region)

		// Warn if region looks invalid
		if region != "" && !isValidRegion(region) {
			fmt.Fprintf(os.Stderr, "%s Region %q is not a common AWS region\n", tui.Yellow.Render("!"), region)
		}

		// Add profile and save
		cfg.AddProfile(config.Profile{
			Name:            profileName,
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
			Region:          region,
		})

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}

		fmt.Fprintf(os.Stderr, "%s Added profile %q (region: %s)\n",
			tui.Green.Render("✔"),
			profileName,
			regionDisplay(region),
		)
		return nil
	},
}

func isValidRegion(region string) bool {
	for _, r := range commonRegions {
		if r == region {
			return true
		}
	}
	// Allow any region matching the pattern xx-xxx-N
	parts := strings.Split(region, "-")
	return len(parts) == 3
}

func regionDisplay(region string) string {
	if region == "" {
		return "not set"
	}
	return region
}
