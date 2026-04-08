package user

import (
	"fmt"
	"os"

	"github.com/isac7722/aws-cli-extension/internal/config"
	"github.com/isac7722/aws-cli-extension/internal/tui"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [profile]",
	Short: "Delete an AWS credential profile",
	Long:  "Delete an AWS credential profile with confirmation. If no profile name is given, an interactive selector is shown.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
	cfg, err := loadAWSConfig()
	if err != nil {
		return err
	}

	if len(cfg.Profiles) == 0 {
		fmt.Fprintln(os.Stderr, tui.Yellow.Render("No profiles found. Run 'awse user add' to create one."))
		return nil
	}

	var profileName string

	if len(args) == 1 {
		profileName = args[0]
		if _, ok := cfg.Get(profileName); !ok {
			return fmt.Errorf("profile %q not found", profileName)
		}
	} else {
		// Interactive selector
		items := profilesToSelectorItems(cfg)
		idx, err := tui.RunSelector(items, "Select profile to delete")
		if err != nil {
			return fmt.Errorf("selector error: %w", err)
		}
		if idx < 0 {
			fmt.Fprintln(os.Stderr, tui.Dim.Render("Cancelled."))
			return nil
		}
		profileName = items[idx].Value
	}

	// Show profile details before confirmation
	p, _ := cfg.Get(profileName)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "  Profile:    %s\n", tui.Bold.Render(p.Name))
	if p.AccessKeyID != "" {
		fmt.Fprintf(os.Stderr, "  Access Key: %s\n", config.MaskKey(p.AccessKeyID))
	}
	if p.Region != "" {
		fmt.Fprintf(os.Stderr, "  Region:     %s\n", p.Region)
	}
	fmt.Fprintln(os.Stderr, "")

	// Confirmation prompt
	confirmed, err := tui.RunConfirm(fmt.Sprintf("Delete profile %q?", profileName), tui.WithDestructive())
	if err != nil {
		return fmt.Errorf("confirm error: %w", err)
	}
	if !confirmed {
		fmt.Fprintln(os.Stderr, tui.Dim.Render("Cancelled."))
		return nil
	}

	// Remove and save
	if !cfg.RemoveProfile(profileName) {
		return fmt.Errorf("profile %q not found", profileName)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save after deletion: %w", err)
	}

	fmt.Fprintln(os.Stderr, tui.Green.Render(fmt.Sprintf("✔ Profile %q deleted.", profileName)))
	return nil
}
