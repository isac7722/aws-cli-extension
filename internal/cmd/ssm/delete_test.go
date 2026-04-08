package ssm

import (
	"testing"
)

func TestDeleteCmd_Flags(t *testing.T) {
	cmd := deleteCmd

	nameFlag := cmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Fatal("expected --name flag to be defined")
	}
}

func TestDeleteCmd_NameRequired(t *testing.T) {
	cmd := deleteCmd
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

func TestDeleteCmd_IsSubcommandOfSSM(t *testing.T) {
	found := false
	for _, sub := range Cmd.Commands() {
		if sub.Use == "delete" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'delete' to be a subcommand of 'ssm'")
	}
}

func TestRunDelete_ValidationErrors(t *testing.T) {
	cmd := deleteCmd

	// Save original and restore after test.
	origName := deleteFlagName
	defer func() { deleteFlagName = origName }()

	deleteFlagName = "no-leading-slash"
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for name without leading slash, got nil")
	}
	if !contains(err.Error(), "parameter name must start with '/'") {
		t.Errorf("error %q does not contain expected message", err.Error())
	}
}
