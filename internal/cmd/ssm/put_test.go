package ssm

import (
	"testing"
)

func TestPutCmd_Flags(t *testing.T) {
	cmd := putCmd

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

func TestPutCmd_IsSubcommandOfSSM(t *testing.T) {
	found := false
	for _, sub := range Cmd.Commands() {
		if sub.Use == "put" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'put' to be a subcommand of 'ssm'")
	}
}

func TestExecutePut_ValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		paramName  string
		value      string
		paramType  string
		wantErrMsg string
	}{
		{
			name:       "missing leading slash",
			paramName:  "app/config/key",
			value:      "val",
			paramType:  "String",
			wantErrMsg: "parameter name must start with '/'",
		},
		{
			name:       "invalid type",
			paramName:  "/app/config/key",
			value:      "val",
			paramType:  "Invalid",
			wantErrMsg: "invalid parameter type",
		},
		{
			name:       "empty value",
			paramName:  "/app/config/key",
			value:      "   ",
			paramType:  "String",
			wantErrMsg: "parameter value must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executePut(tt.paramName, tt.value, tt.paramType, "", false)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
