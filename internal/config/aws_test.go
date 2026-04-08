package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndSaveRoundTrip(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")
	configPath := filepath.Join(tmpDir, "config")

	// Write test credentials file
	credContent := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
region = us-east-1

[production]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY
`
	if err := os.WriteFile(credPath, []byte(credContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Write test config file
	cfgContent := `[default]
region = us-east-1

[profile production]
region = eu-west-1
`
	if err := os.WriteFile(configPath, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Override env vars to use temp paths
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	// Load profiles
	cfg, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(cfg.Profiles))
	}

	// Check default profile
	def, ok := cfg.Get("default")
	if !ok {
		t.Fatal("default profile not found")
	}
	if def.AccessKeyID != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("unexpected access key: %s", def.AccessKeyID)
	}
	if def.Region != "us-east-1" {
		t.Errorf("unexpected region: %s", def.Region)
	}

	// Check production profile
	prod, ok := cfg.Get("production")
	if !ok {
		t.Fatal("production profile not found")
	}
	if prod.AccessKeyID != "AKIAI44QH8DHBEXAMPLE" {
		t.Errorf("unexpected access key: %s", prod.AccessKeyID)
	}
	// Region from config file should be picked up
	if prod.Region != "eu-west-1" {
		t.Errorf("expected region eu-west-1, got %s", prod.Region)
	}
}

func TestUpdateProfile(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")
	configPath := filepath.Join(tmpDir, "config")

	credContent := `[myprofile]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
region = us-east-1
`
	if err := os.WriteFile(credPath, []byte(credContent), 0600); err != nil {
		t.Fatal(err)
	}
	// Empty config
	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	cfg, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	// Update profile
	updated := Profile{
		Name:            "renamed",
		AccessKeyID:     "AKIANEWKEY1234567890",
		SecretAccessKey: "newSecretKey",
		Region:          "ap-northeast-2",
	}

	if !cfg.UpdateProfile("myprofile", updated) {
		t.Fatal("UpdateProfile returned false")
	}

	// Verify old name gone, new name accessible
	if _, ok := cfg.Get("myprofile"); ok {
		t.Error("old profile name should not exist")
	}
	p, ok := cfg.Get("renamed")
	if !ok {
		t.Fatal("renamed profile not found")
	}
	if p.AccessKeyID != "AKIANEWKEY1234567890" {
		t.Errorf("unexpected access key: %s", p.AccessKeyID)
	}

	// Save and reload
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	cfg2, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	p2, ok := cfg2.Get("renamed")
	if !ok {
		t.Fatal("renamed profile not found after reload")
	}
	if p2.Region != "ap-northeast-2" {
		t.Errorf("unexpected region after reload: %s", p2.Region)
	}
}

func TestConfigOnlyProfile(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")
	configPath := filepath.Join(tmpDir, "config")

	// Only default in credentials
	credContent := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`
	// config-only profile has no credentials entry
	cfgContent := `[default]
region = us-east-1

[profile config-only]
region = ap-northeast-2
output = json
`
	if err := os.WriteFile(credPath, []byte(credContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	cfg, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(cfg.Profiles))
	}

	p, ok := cfg.Get("config-only")
	if !ok {
		t.Fatal("config-only profile not found")
	}
	if p.Region != "ap-northeast-2" {
		t.Errorf("expected region ap-northeast-2, got %s", p.Region)
	}
	if p.Output != "json" {
		t.Errorf("expected output json, got %s", p.Output)
	}
	if p.HasCredentials() {
		t.Error("config-only profile should not have credentials")
	}
}

func TestSessionTokenRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")
	configPath := filepath.Join(tmpDir, "config")

	credContent := `[temp]
aws_access_key_id = ASIATEMP12345678EXAM
aws_secret_access_key = tempSecret
aws_session_token = FwoGZXIvYXdzEBYaDH5example
`
	if err := os.WriteFile(credPath, []byte(credContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	cfg, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	p, ok := cfg.Get("temp")
	if !ok {
		t.Fatal("temp profile not found")
	}
	if p.SessionToken != "FwoGZXIvYXdzEBYaDH5example" {
		t.Errorf("unexpected session token: %s", p.SessionToken)
	}

	// Save and reload
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	cfg2, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	p2, ok := cfg2.Get("temp")
	if !ok {
		t.Fatal("temp profile not found after reload")
	}
	if p2.SessionToken != "FwoGZXIvYXdzEBYaDH5example" {
		t.Errorf("session token not preserved after save: %s", p2.SessionToken)
	}
}

func TestProfileNames(t *testing.T) {
	cfg := &AWSConfig{byIdx: make(map[string]int)}
	cfg.AddProfile(Profile{Name: "default", Region: "us-east-1"})
	cfg.AddProfile(Profile{Name: "prod", Region: "eu-west-1"})

	names := cfg.ProfileNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "default" || names[1] != "prod" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestHasCredentials(t *testing.T) {
	p1 := Profile{Name: "full", AccessKeyID: "AKIA...", SecretAccessKey: "secret"}
	if !p1.HasCredentials() {
		t.Error("expected HasCredentials to be true")
	}

	p2 := Profile{Name: "partial", AccessKeyID: "AKIA..."}
	if p2.HasCredentials() {
		t.Error("expected HasCredentials to be false with missing secret")
	}

	p3 := Profile{Name: "empty"}
	if p3.HasCredentials() {
		t.Error("expected HasCredentials to be false for empty profile")
	}
}

func TestMissingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	// Point to non-existent files — should not error
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(tmpDir, "noexist-creds"))
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(tmpDir, "noexist-config"))

	cfg, err := LoadProfiles()
	if err != nil {
		t.Fatalf("expected no error for missing files, got: %v", err)
	}
	if len(cfg.Profiles) != 0 {
		t.Fatalf("expected 0 profiles, got %d", len(cfg.Profiles))
	}
}

func TestCredentialsRegionTakesPriority(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")
	configPath := filepath.Join(tmpDir, "config")

	// Region set in credentials
	credContent := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = secret
region = us-west-2
`
	// Different region in config
	cfgContent := `[default]
region = eu-west-1
`
	if err := os.WriteFile(credPath, []byte(credContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	cfg, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	def, ok := cfg.Get("default")
	if !ok {
		t.Fatal("default profile not found")
	}
	// Credentials region should win
	if def.Region != "us-west-2" {
		t.Errorf("expected region us-west-2 (from credentials), got %s", def.Region)
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AKIAIOSFODNN7EXAMPLE", "AKIA************MPLE"},
		{"short", "****"},
		{"12345678", "****"},
		{"123456789", "1234*6789"},
	}
	for _, tt := range tests {
		got := MaskKey(tt.input)
		if got != tt.expected {
			t.Errorf("MaskKey(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSavePreservesComments(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")
	configPath := filepath.Join(tmpDir, "config")

	// Credentials file with comments
	credContent := `# Main credentials file
# Last updated: 2024-01-01

[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
# This is a comment inside the default section

[production]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY
`
	cfgContent := `[default]
region = us-east-1

[profile production]
region = eu-west-1
`
	if err := os.WriteFile(credPath, []byte(credContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	cfg, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	// Update default profile region only
	def, _ := cfg.Get("default")
	def.Region = "us-west-2"
	cfg.UpdateProfile("default", *def)

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// Read saved credentials and verify comments are preserved
	savedCred, err := os.ReadFile(credPath)
	if err != nil {
		t.Fatal(err)
	}
	credStr := string(savedCred)

	if !strings.Contains(credStr, "# Main credentials file") {
		t.Error("preamble comment was not preserved in credentials file")
	}
	if !strings.Contains(credStr, "# This is a comment inside the default section") {
		t.Error("inline comment was not preserved in credentials file")
	}
	// Production profile should still be there
	if !strings.Contains(credStr, "AKIAI44QH8DHBEXAMPLE") {
		t.Error("production profile credentials were lost")
	}
}

func TestSavePreservesUnknownKeys(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")
	configPath := filepath.Join(tmpDir, "config")

	// Credentials with unknown keys (source_profile, role_arn, mfa_serial)
	credContent := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[cross-account]
role_arn = arn:aws:iam::123456789012:role/MyRole
source_profile = default
mfa_serial = arn:aws:iam::123456789012:mfa/user
`
	cfgContent := `[default]
region = us-east-1
output = json
cli_pager =

[profile cross-account]
region = ap-northeast-2
s3 =
  max_concurrent_requests = 20
`
	if err := os.WriteFile(credPath, []byte(credContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	cfg, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	// Just update default's access key
	def, _ := cfg.Get("default")
	def.AccessKeyID = "AKIANEWKEY1234567890"
	cfg.UpdateProfile("default", *def)

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// Verify unknown keys are preserved
	savedCred, err := os.ReadFile(credPath)
	if err != nil {
		t.Fatal(err)
	}
	credStr := string(savedCred)

	if !strings.Contains(credStr, "role_arn = arn:aws:iam::123456789012:role/MyRole") {
		t.Error("role_arn was not preserved in credentials file")
	}
	if !strings.Contains(credStr, "source_profile = default") {
		t.Error("source_profile was not preserved in credentials file")
	}
	if !strings.Contains(credStr, "mfa_serial = arn:aws:iam::123456789012:mfa/user") {
		t.Error("mfa_serial was not preserved in credentials file")
	}

	// Verify the updated key
	if !strings.Contains(credStr, "AKIANEWKEY1234567890") {
		t.Error("updated access key not found")
	}

	// Verify config unknowns preserved
	savedCfg, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfgStr := string(savedCfg)

	if !strings.Contains(cfgStr, "cli_pager") {
		t.Error("cli_pager was not preserved in config file")
	}
	if !strings.Contains(cfgStr, "max_concurrent_requests") {
		t.Error("s3 nested config was not preserved in config file")
	}
}

func TestSaveDeleteProfilePreservesOthers(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")
	configPath := filepath.Join(tmpDir, "config")

	credContent := `[default]
aws_access_key_id = AKIADEFAULT123456789
aws_secret_access_key = defaultSecret

[staging]
aws_access_key_id = AKIASTAGING12345678
aws_secret_access_key = stagingSecret

[production]
aws_access_key_id = AKIAPRODUCT12345678
aws_secret_access_key = productionSecret
`
	cfgContent := `[default]
region = us-east-1

[profile staging]
region = us-west-1

[profile production]
region = eu-west-1
`
	if err := os.WriteFile(credPath, []byte(credContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	cfg, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	// Delete staging profile
	if !cfg.RemoveProfile("staging") {
		t.Fatal("RemoveProfile returned false")
	}

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// Reload and verify
	cfg2, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg2.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(cfg2.Profiles))
	}

	if _, ok := cfg2.Get("staging"); ok {
		t.Error("staging profile should have been deleted")
	}

	// Default and production should still exist with correct data
	def, ok := cfg2.Get("default")
	if !ok {
		t.Fatal("default profile missing after delete")
	}
	if def.AccessKeyID != "AKIADEFAULT123456789" {
		t.Errorf("default access key corrupted: %s", def.AccessKeyID)
	}

	prod, ok := cfg2.Get("production")
	if !ok {
		t.Fatal("production profile missing after delete")
	}
	if prod.Region != "eu-west-1" {
		t.Errorf("production region corrupted: %s", prod.Region)
	}
}

func TestSaveAddProfilePreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")
	configPath := filepath.Join(tmpDir, "config")

	credContent := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`
	cfgContent := `[default]
region = us-east-1
output = json
`
	if err := os.WriteFile(credPath, []byte(credContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	cfg, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	// Add a new profile
	cfg.AddProfile(Profile{
		Name:            "newprofile",
		AccessKeyID:     "AKIANEWPROFILE12345",
		SecretAccessKey: "newProfileSecret",
		Region:          "ap-southeast-1",
	})

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// Reload and verify both profiles exist
	cfg2, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg2.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(cfg2.Profiles))
	}

	// Original profile preserved
	def, ok := cfg2.Get("default")
	if !ok {
		t.Fatal("default profile missing")
	}
	if def.AccessKeyID != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("default access key corrupted: %s", def.AccessKeyID)
	}
	if def.Output != "json" {
		t.Errorf("default output not preserved: %s", def.Output)
	}

	// New profile present
	newP, ok := cfg2.Get("newprofile")
	if !ok {
		t.Fatal("newprofile not found")
	}
	if newP.AccessKeyID != "AKIANEWPROFILE12345" {
		t.Errorf("new profile access key wrong: %s", newP.AccessKeyID)
	}
	if newP.Region != "ap-southeast-1" {
		t.Errorf("new profile region wrong: %s", newP.Region)
	}
}

func TestSaveRenameProfilePreservesOthers(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")
	configPath := filepath.Join(tmpDir, "config")

	credContent := `[default]
aws_access_key_id = AKIADEFAULT123456789
aws_secret_access_key = defaultSecret

[oldname]
aws_access_key_id = AKIAOLDNAME123456789
aws_secret_access_key = oldnameSecret
`
	cfgContent := `[default]
region = us-east-1

[profile oldname]
region = eu-west-1
output = table
`
	if err := os.WriteFile(credPath, []byte(credContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	cfg, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	// Rename oldname -> newname
	updated := Profile{
		Name:            "newname",
		AccessKeyID:     "AKIAOLDNAME123456789",
		SecretAccessKey: "oldnameSecret",
		Region:          "eu-west-1",
		Output:          "table",
	}
	if !cfg.UpdateProfile("oldname", updated) {
		t.Fatal("UpdateProfile returned false")
	}

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// Reload and verify
	cfg2, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := cfg2.Get("oldname"); ok {
		t.Error("old profile name should not exist after rename")
	}

	p, ok := cfg2.Get("newname")
	if !ok {
		t.Fatal("renamed profile not found")
	}
	if p.Region != "eu-west-1" {
		t.Errorf("renamed profile region wrong: %s", p.Region)
	}

	// Default should be untouched
	def, ok := cfg2.Get("default")
	if !ok {
		t.Fatal("default profile missing after rename")
	}
	if def.AccessKeyID != "AKIADEFAULT123456789" {
		t.Errorf("default access key corrupted: %s", def.AccessKeyID)
	}
}

func TestSaveEmptyFilesCreatesNew(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "subdir", "credentials")
	configPath := filepath.Join(tmpDir, "subdir", "config")

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath)
	t.Setenv("AWS_CONFIG_FILE", configPath)

	cfg := &AWSConfig{byIdx: make(map[string]int)}
	cfg.AddProfile(Profile{
		Name:            "default",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "secret",
		Region:          "us-east-1",
		Output:          "json",
	})

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// Verify files were created
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		t.Fatal("credentials file was not created")
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Reload and verify
	cfg2, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg2.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(cfg2.Profiles))
	}
	def, ok := cfg2.Get("default")
	if !ok {
		t.Fatal("default profile not found")
	}
	if def.AccessKeyID != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("access key wrong: %s", def.AccessKeyID)
	}
	if def.Output != "json" {
		t.Errorf("output not preserved: %s", def.Output)
	}
}
