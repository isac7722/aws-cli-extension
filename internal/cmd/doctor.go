package cmd

import (
	"fmt"
	"os"

	"github.com/isac7722/aws-cli-extension/internal/doctor"
	"github.com/isac7722/aws-cli-extension/internal/tui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment health (AWS CLI, credentials, etc.)",
	RunE: func(cmd *cobra.Command, args []string) error {
		results := []doctor.CheckResult{
			doctor.CheckAWSCLI(),
		}

		allOK := true
		for _, r := range results {
			icon := statusIcon(r.Status)
			fmt.Fprintf(os.Stderr, "%s %s: %s\n", icon, tui.Bold.Render(r.Name), r.Message)
			if r.Status != doctor.StatusOK {
				allOK = false
			}
		}

		if !allOK {
			fmt.Fprintln(os.Stderr, "")
			guidance := doctor.GetInstallGuidance()
			fmt.Fprintln(os.Stderr, tui.Yellow.Render(guidance.Title))
			for _, step := range guidance.Steps {
				fmt.Fprintf(os.Stderr, "  %s\n", step)
			}
			fmt.Fprintf(os.Stderr, "\n  Docs: %s\n", tui.Cyan.Render(guidance.URL))
		}
		return nil
	},
}

func statusIcon(s doctor.Status) string {
	switch s {
	case doctor.StatusOK:
		return tui.Green.Render("✓")
	case doctor.StatusWarning:
		return tui.Yellow.Render("!")
	case doctor.StatusError:
		return tui.Red.Render("✗")
	default:
		return "?"
	}
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
