package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestInitBash(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"init", "bash"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Must contain the wrapper function definition
	if !strings.Contains(output, "awse()") {
		t.Error("bash output should contain awse() function definition")
	}

	// Must handle AWSE_EXPORT protocol
	if !strings.Contains(output, "AWSE_EXPORT:") {
		t.Error("bash output should contain AWSE_EXPORT protocol handling")
	}

	// Must contain export command for variable setting
	if !strings.Contains(output, "export") {
		t.Error("bash output should contain export command")
	}

	// Must reference 'user switch' handling
	if !strings.Contains(output, "switch") {
		t.Error("bash output should handle 'user switch' subcommand")
	}
}

func TestInitZsh(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"init", "zsh"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "awse()") {
		t.Error("zsh output should contain awse() function definition")
	}
}

func TestInitUnsupportedShell(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init", "fish"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported shell")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("expected 'unsupported shell' error, got: %v", err)
	}
}

func TestInitNoArgs(t *testing.T) {
	rootCmd.SetArgs([]string{"init"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when no shell argument provided")
	}
}

func TestShellWrapperContainsExportProtocol(t *testing.T) {
	output := shellWrapper("bash")
	if !strings.Contains(output, "AWSE_EXPORT") {
		t.Error("wrapper should contain AWSE_EXPORT protocol for variable export")
	}
	if !strings.Contains(output, "export") {
		t.Error("wrapper should contain export command")
	}
}

func TestShellWrapperPassthroughForNonSwitch(t *testing.T) {
	output := shellWrapper("bash")
	// Non-switch user subcommands should use 'command' passthrough
	if !strings.Contains(output, `command "$awse_bin" "$@"`) {
		t.Error("wrapper should pass through non-switch commands via 'command'")
	}
}

func TestShellWrapperParsesKeyValue(t *testing.T) {
	output := shellWrapper("bash")
	// Wrapper must extract key and value from AWSE_EXPORT:KEY=VALUE
	if !strings.Contains(output, `${line#AWSE_EXPORT:}`) {
		t.Error("wrapper should strip AWSE_EXPORT: prefix to get assignment")
	}
	if !strings.Contains(output, `${assignment%%=*}`) {
		t.Error("wrapper should extract key from assignment")
	}
	if !strings.Contains(output, `${assignment#*=}`) {
		t.Error("wrapper should extract value from assignment")
	}
}
