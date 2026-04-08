package ssm

import (
	"testing"
)

func TestBatchDeleteCmd_Flags(t *testing.T) {
	cmd := batchDeleteCmd

	nameFlag := cmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Fatal("expected --name flag to be defined")
	}
}

func TestBatchDeleteCmd_NameRequired(t *testing.T) {
	cmd := batchDeleteCmd
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

func TestBatchDeleteCmd_IsSubcommandOfSSM(t *testing.T) {
	found := false
	for _, sub := range Cmd.Commands() {
		if sub.Use == "batch-delete" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'batch-delete' to be a subcommand of 'ssm'")
	}
}

func TestRunBatchDelete_ValidationErrors(t *testing.T) {
	cmd := batchDeleteCmd

	// Save original and restore after test.
	origNames := batchDeleteFlagNames
	defer func() { batchDeleteFlagNames = origNames }()

	batchDeleteFlagNames = []string{"/valid/param", "no-leading-slash"}
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for name without leading slash, got nil")
	}
	if !contains(err.Error(), "parameter name must start with '/'") {
		t.Errorf("error %q does not contain expected message", err.Error())
	}
}

func TestDedup(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"/a", "/b", "/c"},
			expected: []string{"/a", "/b", "/c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"/a", "/b", "/a", "/c", "/b"},
			expected: []string{"/a", "/b", "/c"},
		},
		{
			name:     "all same",
			input:    []string{"/a", "/a", "/a"},
			expected: []string{"/a"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dedup(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("dedup(%v) returned %d items, want %d", tt.input, len(result), len(tt.expected))
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("dedup(%v)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}
