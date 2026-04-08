package aws

import (
	"github.com/isac7722/aws-cli-extension/internal/config"
)

// ListProfiles returns profile names from the merged credentials and config files.
// This is a convenience wrapper around config.LoadProfiles.
func ListProfiles() ([]string, error) {
	cfg, err := config.LoadProfiles()
	if err != nil {
		return nil, err
	}
	return cfg.ProfileNames(), nil
}
