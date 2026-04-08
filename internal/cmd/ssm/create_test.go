package ssm

import (
	"testing"
)

func TestIsValidType(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"String", true},
		{"StringList", true},
		{"SecureString", true},
		{"string", false},
		{"", false},
		{"Invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isValidType(tt.input)
			if got != tt.want {
				t.Errorf("isValidType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCreateCmd_RequiredFlags(t *testing.T) {
	// Reset flags for test isolation
	cmd := createCmd

	// Test that the command has the expected flags
	nameFlag := cmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Fatal("expected --name flag to be defined")
	}

	valueFlag := cmd.Flags().Lookup("value")
	if valueFlag == nil {
		t.Fatal("expected --value flag to be defined")
	}

	typeFlag := cmd.Flags().Lookup("type")
	if typeFlag == nil {
		t.Fatal("expected --type flag to be defined")
	}
	if typeFlag.DefValue != "String" {
		t.Errorf("--type default = %q, want %q", typeFlag.DefValue, "String")
	}

	descFlag := cmd.Flags().Lookup("description")
	if descFlag == nil {
		t.Fatal("expected --description flag to be defined")
	}

	overwriteFlag := cmd.Flags().Lookup("overwrite")
	if overwriteFlag == nil {
		t.Fatal("expected --overwrite flag to be defined")
	}
	if overwriteFlag.DefValue != "false" {
		t.Errorf("--overwrite default = %q, want %q", overwriteFlag.DefValue, "false")
	}
}

func TestCreateCmd_IsSubcommandOfSSM(t *testing.T) {
	found := false
	for _, sub := range Cmd.Commands() {
		if sub.Use == "create" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'create' to be a subcommand of 'ssm'")
	}
}
