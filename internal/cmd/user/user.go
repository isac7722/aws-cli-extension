package user

import (
	"fmt"
	"os"

	"github.com/isac7722/aws-cli-extension/internal/config"
	"github.com/isac7722/aws-cli-extension/internal/tui"
	"github.com/spf13/cobra"
)

// Cmd is the user parent command.
// When called without a subcommand, it launches the interactive profile switcher.
var Cmd = &cobra.Command{
	Use:   "user",
	Short: "Manage AWS credential profiles",
	Long:  "List, add, edit, delete, and switch between AWS credential profiles.\nRun without a subcommand to interactively switch profiles.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInteractiveSwitch(cmd)
	},
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(addCmd)
	Cmd.AddCommand(editCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(switchCmd)
}

// runSelector is a variable so tests can replace it.
var runSelector = tui.RunSelector

func loadAWSConfig() (*config.AWSConfig, error) {
	cfg, err := config.LoadProfiles()
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS profiles: %w\nRun 'awse user add' to create your first profile", err)
	}
	return cfg, nil
}

func profilesToSelectorItems(cfg *config.AWSConfig) []tui.SelectorItem {
	activeProfile := activeAWSProfile()
	var items []tui.SelectorItem
	for _, p := range cfg.Profiles {
		selected := p.Name == activeProfile
		hint := ""
		if p.AccessKeyID != "" {
			hint = fmt.Sprintf("%s  %s", config.MaskKey(p.AccessKeyID), p.Region)
		} else if p.Region != "" {
			hint = p.Region
		}
		items = append(items, tui.SelectorItem{
			Label:    p.Name,
			Value:    p.Name,
			Hint:     hint,
			Selected: selected,
		})
	}
	return items
}

func activeAWSProfile() string {
	return getenv("AWS_PROFILE")
}

// runInteractiveSwitch loads profiles, shows the TUI selector, and outputs
// the AWSE_EXPORT protocol line. Used by both `awse user` and `awse user switch`.
func runInteractiveSwitch(cmd *cobra.Command) error {
	cfg, err := loadAWSConfig()
	if err != nil {
		return err
	}

	if len(cfg.Profiles) == 0 {
		fmt.Println("No AWS profiles configured. Run 'awse user add' to create one.")
		return nil
	}

	items := profilesToSelectorItems(cfg)

	chosen, err := runSelector(items, "Switch AWS profile")
	if err != nil {
		return fmt.Errorf("selector error: %w", err)
	}

	if chosen < 0 {
		return nil // cancelled
	}

	profileName := items[chosen].Value

	// Find the full profile for the card display
	var profile *config.Profile
	for i, p := range cfg.Profiles {
		if p.Name == profileName {
			profile = &cfg.Profiles[i]
			break
		}
	}

	// Output AWSE_EXPORT protocol for shell wrapper
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "AWSE_EXPORT:AWS_PROFILE=%s\n", profileName)

	// Profile card to stderr
	fmt.Fprintf(os.Stderr, "\n%s %s\n", tui.Green.Render("✔"), tui.Bold.Render("Switched to "+profileName))

	if profile != nil {
		var rows []string
		rows = append(rows, tui.CardLabel.Render("Profile")+" "+tui.Bold.Render(profile.Name))
		if profile.Region != "" {
			rows = append(rows, tui.CardLabel.Render("Region")+" "+profile.Region)
		}
		if profile.AccessKeyID != "" {
			rows = append(rows, tui.CardLabel.Render("Key ID")+" "+config.MaskKey(profile.AccessKeyID))
		}
		if profile.Output != "" {
			rows = append(rows, tui.CardLabel.Render("Output")+" "+profile.Output)
		}

		card := tui.CardStyle.Render(joinLines(rows))
		fmt.Fprintf(os.Stderr, "%s\n", card)
	}

	return nil
}

func joinLines(lines []string) string {
	result := ""
	for i, l := range lines {
		if i > 0 {
			result += "\n"
		}
		result += l
	}
	return result
}
