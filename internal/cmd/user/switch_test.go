package user

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/isac7722/aws-cli-extension/internal/tui"
)

func TestSwitchCmd_SelectsProfile(t *testing.T) {
	origSelector := runSelector
	defer func() { runSelector = origSelector }()

	var capturedItems []tui.SelectorItem
	runSelector = func(items []tui.SelectorItem, header string) (int, error) {
		capturedItems = items
		return 1, nil
	}

	dir := t.TempDir()
	setupTestAWSFiles(t, dir,
		"[default]\naws_access_key_id = AKIA1111\naws_secret_access_key = secret1\n\n[production]\naws_access_key_id = AKIA2222\naws_secret_access_key = secret2\n",
		"[default]\nregion = us-east-1\n\n[profile production]\nregion = ap-northeast-2\n",
	)

	var buf bytes.Buffer
	switchCmd.SetOut(&buf)
	defer switchCmd.SetOut(nil)

	err := switchCmd.RunE(switchCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedItems) != 2 {
		t.Fatalf("expected 2 items, got %d", len(capturedItems))
	}
	if capturedItems[0].Value != "default" {
		t.Errorf("expected first item 'default', got %q", capturedItems[0].Value)
	}
	if capturedItems[1].Value != "production" {
		t.Errorf("expected second item 'production', got %q", capturedItems[1].Value)
	}

	// Verify the AWSE_EXPORT protocol output
	output := buf.String()
	expected := "AWSE_EXPORT:AWS_PROFILE=production"
	if !strings.Contains(output, expected) {
		t.Errorf("expected output to contain %q, got %q", expected, output)
	}
}

func TestSwitchCmd_ExportProtocolFormat(t *testing.T) {
	origSelector := runSelector
	defer func() { runSelector = origSelector }()

	runSelector = func(items []tui.SelectorItem, header string) (int, error) {
		return 0, nil
	}

	dir := t.TempDir()
	setupTestAWSFiles(t, dir,
		"[my-profile]\naws_access_key_id = AKIA1111\naws_secret_access_key = secret1\n",
		"",
	)

	var buf bytes.Buffer
	switchCmd.SetOut(&buf)
	defer switchCmd.SetOut(nil)

	err := switchCmd.RunE(switchCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "AWSE_EXPORT:AWS_PROFILE=my-profile" {
		t.Errorf("expected exact export line 'AWSE_EXPORT:AWS_PROFILE=my-profile', got %q", output)
	}
}

func TestSwitchCmd_Cancelled(t *testing.T) {
	origSelector := runSelector
	defer func() { runSelector = origSelector }()

	runSelector = func(items []tui.SelectorItem, header string) (int, error) {
		return -1, nil
	}

	dir := t.TempDir()
	setupTestAWSFiles(t, dir,
		"[default]\naws_access_key_id = AKIA1111\naws_secret_access_key = secret1\n",
		"",
	)

	err := switchCmd.RunE(switchCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error on cancel: %v", err)
	}
}

func TestSwitchCmd_NoProfiles(t *testing.T) {
	dir := t.TempDir()
	setupTestAWSFiles(t, dir, "", "")

	err := switchCmd.RunE(switchCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error with no profiles: %v", err)
	}
}

func setupTestAWSFiles(t *testing.T, dir, credentials, config string) {
	t.Helper()
	credFile := dir + "/credentials"
	configFile := dir + "/config"
	if err := os.WriteFile(credFile, []byte(credentials), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configFile, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credFile)
	t.Setenv("AWS_CONFIG_FILE", configFile)
	t.Setenv("AWS_PROFILE", "")
}
