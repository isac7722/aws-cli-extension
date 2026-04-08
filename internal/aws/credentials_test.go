package aws

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")
	configPath := filepath.Join(tmpDir, "config")

	credContent := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[staging]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = secret2
`
	cfgContent := `[default]
region = us-east-1

[profile staging]
region = eu-west-1

[profile config-only]
region = ap-northeast-2
`
	if err := os.WriteFile(credPath, []byte(credContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	profiles, err := ListProfiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include profiles from both files: default, staging, config-only
	if len(profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d: %v", len(profiles), profiles)
	}

	expected := []string{"default", "staging", "config-only"}
	for i, name := range expected {
		if profiles[i] != name {
			t.Errorf("profile[%d]: expected %q, got %q", i, name, profiles[i])
		}
	}
}

func TestListProfiles_EmptyFiles(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")
	configPath := filepath.Join(tmpDir, "config")

	if err := os.WriteFile(credPath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	profiles, err := ListProfiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(profiles) != 0 {
		t.Fatalf("expected 0 profiles, got %d: %v", len(profiles), profiles)
	}
}
