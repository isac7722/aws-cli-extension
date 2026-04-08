package user

import (
	"fmt"

	"github.com/isac7722/aws-cli-extension/internal/config"
	"github.com/isac7722/aws-cli-extension/internal/tui"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:     "edit [profile]",
	Short:   "Edit an existing AWS credential profile",
	Aliases: []string{"update"},
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadAWSConfig()
		if err != nil {
			return err
		}

		if len(cfg.Profiles) == 0 {
			fmt.Println("No AWS profiles configured. Run 'awse user add' to create one.")
			return nil
		}

		var profileName string
		var profile *config.Profile

		if len(args) == 1 {
			profileName = args[0]
			p, ok := cfg.Get(profileName)
			if !ok {
				return fmt.Errorf("profile %q not found", profileName)
			}
			profile = p
		} else {
			// Show TUI selector to choose profile
			items := profilesToSelectorItems(cfg)
			idx, err := tui.RunSelector(items, "Select profile to edit:")
			if err != nil {
				return err
			}
			if idx < 0 {
				return nil // cancelled
			}
			profileName = cfg.Profiles[idx].Name
			profile = &cfg.Profiles[idx]
		}

		// Profile name
		newName, ok, err := tui.RunPromptWithValue("Profile name:", "e.g., production", profileName)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelled.")
			return nil
		}
		if newName == "" {
			newName = profileName
		}
		if newName != profileName {
			if _, exists := cfg.Get(newName); exists {
				return fmt.Errorf("profile %q already exists", newName)
			}
		}

		// Access Key ID
		accessKeyID, ok, err := tui.RunPromptWithValue("AWS Access Key ID:", "AKIA...", profile.AccessKeyID)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelled.")
			return nil
		}
		if accessKeyID == "" {
			accessKeyID = profile.AccessKeyID
		}

		// Secret Access Key (masked input)
		secretPromptLabel := "AWS Secret Access Key:"
		secretPlaceholder := "wJalr..."
		if profile.SecretAccessKey != "" {
			secretPlaceholder = "(press Enter to keep current)"
		}
		secretAccessKey, ok, err := tui.RunSecretPrompt(secretPromptLabel, secretPlaceholder)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelled.")
			return nil
		}
		if secretAccessKey == "" {
			secretAccessKey = profile.SecretAccessKey
		}

		// Region
		region, ok, err := tui.RunPromptWithValue("Default region:", "e.g., us-east-1", profile.Region)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelled.")
			return nil
		}
		if region == "" {
			region = profile.Region
		}

		updated := config.Profile{
			Name:            newName,
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
			Region:          region,
		}

		cfg.UpdateProfile(profileName, updated)
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}

		fmt.Printf("✔ Updated profile %q (region: %s)\n", newName, region)
		return nil
	},
}
