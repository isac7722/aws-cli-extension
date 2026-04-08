package ssm

import (
	"testing"
)

func TestUpdateCmd_Flags(t *testing.T) {
	cmd := updateCmd

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
	// update command defaults to empty type (inherits from existing parameter)
	if typeFlag.DefValue != "" {
		t.Errorf("--type default = %q, want %q", typeFlag.DefValue, "")
	}

	descFlag := cmd.Flags().Lookup("description")
	if descFlag == nil {
		t.Fatal("expected --description flag to be defined")
	}

	overwriteFlag := cmd.Flags().Lookup("overwrite")
	if overwriteFlag == nil {
		t.Fatal("expected --overwrite flag to be defined")
	}
	// update command defaults overwrite to true (unlike create/put which default to false)
	if overwriteFlag.DefValue != "true" {
		t.Errorf("--overwrite default = %q, want %q", overwriteFlag.DefValue, "true")
	}
}

func TestUpdateCmd_IsSubcommandOfSSM(t *testing.T) {
	found := false
	for _, sub := range Cmd.Commands() {
		if sub.Use == "update" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'update' to be a subcommand of 'ssm'")
	}
}

func TestExecuteUpdate_ValidationErrors(t *testing.T) {
	// Create a minimal cobra command to simulate flag state for executeUpdate.
	// We use the actual updateCmd since executeUpdate checks cmd.Flags().Changed("type").
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
			// We need a command with the type flag marked as changed when paramType is set.
			// For validation-only tests, executeUpdate will fail before needing AWS.
			err := executeUpdate(updateCmd, tt.paramName, tt.value, tt.paramType, "", true)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func TestUpdateCmd_OverwriteDefaultsTrue(t *testing.T) {
	// Verify that the update command's overwrite flag defaults to true,
	// which is the key behavioral difference from create/put.
	flag := updateCmd.Flags().Lookup("overwrite")
	if flag == nil {
		t.Fatal("expected --overwrite flag")
	}
	if flag.DefValue != "true" {
		t.Errorf("update --overwrite default = %q, want %q", flag.DefValue, "true")
	}
}

func TestUpdateCmd_TypeDefaultsEmpty(t *testing.T) {
	// Verify that the update command's type flag defaults to empty,
	// meaning the existing parameter's type is preserved unless explicitly changed.
	flag := updateCmd.Flags().Lookup("type")
	if flag == nil {
		t.Fatal("expected --type flag")
	}
	if flag.DefValue != "" {
		t.Errorf("update --type default = %q, want empty string", flag.DefValue)
	}
}
