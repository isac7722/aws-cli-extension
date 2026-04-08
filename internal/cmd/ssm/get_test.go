package ssm

import (
	"testing"
)

func TestGetCmd_Flags(t *testing.T) {
	cmd := getCmd

	nameFlag := cmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Fatal("expected --name flag to be defined")
	}

	decryptFlag := cmd.Flags().Lookup("decrypt")
	if decryptFlag == nil {
		t.Fatal("expected --decrypt flag to be defined")
	}
	if decryptFlag.DefValue != "false" {
		t.Errorf("--decrypt default = %q, want %q", decryptFlag.DefValue, "false")
	}
}

func TestGetCmd_NameRequired(t *testing.T) {
	// The --name flag should be marked as required.
	cmd := getCmd
	nameFlag := cmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Fatal("expected --name flag to be defined")
	}

	// Cobra marks required flags with an annotation.
	annotations := nameFlag.Annotations
	if annotations == nil {
		t.Fatal("expected --name flag to have annotations (required)")
	}
	required, ok := annotations["cobra_annotation_bash_completion_one_required_flag"]
	if !ok || len(required) == 0 {
		t.Error("expected --name flag to be marked as required")
	}
}

func TestGetCmd_IsSubcommandOfSSM(t *testing.T) {
	found := false
	for _, sub := range Cmd.Commands() {
		if sub.Use == "get" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'get' to be a subcommand of 'ssm'")
	}
}

func TestRunGet_ValidationErrors(t *testing.T) {
	// Test that runGet rejects names without leading slash.
	// We need to set the flag value directly and invoke RunE.
	cmd := getCmd

	// Save original and restore after test.
	origName := getFlagName
	defer func() { getFlagName = origName }()

	getFlagName = "no-leading-slash"
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for name without leading slash, got nil")
	}
	if !contains(err.Error(), "parameter name must start with '/'") {
		t.Errorf("error %q does not contain expected message", err.Error())
	}
}
