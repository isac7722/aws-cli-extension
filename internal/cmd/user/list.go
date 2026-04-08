package user

import (
	"fmt"

	"github.com/isac7722/aws-cli-extension/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all AWS credential profiles",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadAWSConfig()
		if err != nil {
			return err
		}

		if len(cfg.Profiles) == 0 {
			fmt.Println("No AWS profiles configured. Run 'awse user add' to create one.")
			return nil
		}

		activeProfile := activeAWSProfile()
		for _, p := range cfg.Profiles {
			marker := "  "
			if p.Name == activeProfile {
				marker = "* "
			}
			hint := ""
			if p.AccessKeyID != "" {
				hint = fmt.Sprintf("  %s  %s", config.MaskKey(p.AccessKeyID), p.Region)
			} else if p.Region != "" {
				hint = fmt.Sprintf("  %s", p.Region)
			}
			fmt.Printf("%s%s%s\n", marker, p.Name, hint)
		}
		return nil
	},
}
