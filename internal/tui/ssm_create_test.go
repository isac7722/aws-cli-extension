package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSSMCreate_DefaultValues(t *testing.T) {
	m := NewSSMCreate("Create Parameter", SSMCreateResult{})

	if m.header != "Create Parameter" {
		t.Errorf("header = %q, want %q", m.header, "Create Parameter")
	}
	if m.focused != ssmFieldName {
		t.Errorf("focused = %d, want %d (ssmFieldName)", m.focused, ssmFieldName)
	}
	if m.typeIndex != 0 {
		t.Errorf("typeIndex = %d, want 0 (String)", m.typeIndex)
	}
	if m.done || m.quit {
		t.Error("form should not be done or quit on creation")
	}
	if len(m.inputs) != int(ssmFieldCount) {
		t.Errorf("input count = %d, want %d", len(m.inputs), ssmFieldCount)
	}
}

func TestNewSSMCreate_WithInitialValues(t *testing.T) {
	initial := SSMCreateResult{
		Name:        "/app/config/test",
		Value:       "hello",
		Type:        "SecureString",
		Description: "test param",
	}
	m := NewSSMCreate("Edit Parameter", initial)

	if m.inputs[ssmFieldName].Value() != "/app/config/test" {
		t.Errorf("name = %q, want %q", m.inputs[ssmFieldName].Value(), "/app/config/test")
	}
	if m.inputs[ssmFieldValue].Value() != "hello" {
		t.Errorf("value = %q, want %q", m.inputs[ssmFieldValue].Value(), "hello")
	}
	if m.typeIndex != 2 {
		t.Errorf("typeIndex = %d, want 2 (SecureString)", m.typeIndex)
	}
	if m.inputs[ssmFieldDescription].Value() != "test param" {
		t.Errorf("description = %q, want %q", m.inputs[ssmFieldDescription].Value(), "test param")
	}
}

func TestSSMCreateModel_FocusNavigation(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})

	// Initial focus on name field.
	if m.focused != ssmFieldName {
		t.Fatalf("initial focused = %d, want %d", m.focused, ssmFieldName)
	}

	// Tab moves to next field.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(SSMCreateModel)
	if m.focused != ssmFieldValue {
		t.Errorf("after tab: focused = %d, want %d (ssmFieldValue)", m.focused, ssmFieldValue)
	}

	// Tab again to type.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(SSMCreateModel)
	if m.focused != ssmFieldType {
		t.Errorf("after 2nd tab: focused = %d, want %d (ssmFieldType)", m.focused, ssmFieldType)
	}

	// Shift+tab back to value.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(SSMCreateModel)
	if m.focused != ssmFieldValue {
		t.Errorf("after shift+tab: focused = %d, want %d (ssmFieldValue)", m.focused, ssmFieldValue)
	}
}

func TestSSMCreateModel_TypeCycling(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})

	// Navigate to type field.
	m.focused = ssmFieldType
	if ssmParamTypes[m.typeIndex] != "String" {
		t.Fatalf("initial type = %q, want %q", ssmParamTypes[m.typeIndex], "String")
	}

	// Right arrow cycles forward.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(SSMCreateModel)
	if ssmParamTypes[m.typeIndex] != "StringList" {
		t.Errorf("after right: type = %q, want %q", ssmParamTypes[m.typeIndex], "StringList")
	}

	// Right again.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(SSMCreateModel)
	if ssmParamTypes[m.typeIndex] != "SecureString" {
		t.Errorf("after 2nd right: type = %q, want %q", ssmParamTypes[m.typeIndex], "SecureString")
	}

	// Right wraps around.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(SSMCreateModel)
	if ssmParamTypes[m.typeIndex] != "String" {
		t.Errorf("after 3rd right: type = %q, want %q", ssmParamTypes[m.typeIndex], "String")
	}

	// Left wraps backward.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(SSMCreateModel)
	if ssmParamTypes[m.typeIndex] != "SecureString" {
		t.Errorf("after left: type = %q, want %q", ssmParamTypes[m.typeIndex], "SecureString")
	}
}

func TestSSMCreateModel_Cancel(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(SSMCreateModel)

	if !m.Cancelled() {
		t.Error("expected Cancelled() to return true")
	}
	if m.Result() != nil {
		t.Error("expected Result() to return nil on cancel")
	}
	if cmd == nil {
		t.Error("expected quit command on esc")
	}
}

func TestSSMCreateModel_CtrlCCancel(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(SSMCreateModel)

	if !m.Cancelled() {
		t.Error("expected Cancelled() to return true on ctrl+c")
	}
}

func TestSSMCreateModel_Validate_EmptyName(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})
	err := m.validate()
	if err == "" {
		t.Error("expected validation error for empty name")
	}
	if !strings.Contains(err, "name is required") {
		t.Errorf("error = %q, want it to contain 'name is required'", err)
	}
}

func TestSSMCreateModel_Validate_MissingSlash(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})
	m.inputs[ssmFieldName].SetValue("no-slash")
	m.inputs[ssmFieldValue].SetValue("test")

	err := m.validate()
	if err == "" {
		t.Error("expected validation error for missing slash")
	}
	if !strings.Contains(err, "must start with '/'") {
		t.Errorf("error = %q, want it to contain \"must start with '/'\"", err)
	}
}

func TestSSMCreateModel_Validate_EmptyValue(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})
	m.inputs[ssmFieldName].SetValue("/app/test")

	err := m.validate()
	if err == "" {
		t.Error("expected validation error for empty value")
	}
	if !strings.Contains(err, "value is required") {
		t.Errorf("error = %q, want it to contain 'value is required'", err)
	}
}

func TestSSMCreateModel_Validate_Success(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})
	m.inputs[ssmFieldName].SetValue("/app/config/key")
	m.inputs[ssmFieldValue].SetValue("my-value")

	err := m.validate()
	if err != "" {
		t.Errorf("expected no validation error, got %q", err)
	}
}

func TestSSMCreateModel_Result(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})
	m.inputs[ssmFieldName].SetValue("/app/config/key")
	m.inputs[ssmFieldValue].SetValue("my-value")
	m.inputs[ssmFieldDescription].SetValue("test desc")
	m.typeIndex = 2 // SecureString
	m.done = true

	result := m.Result()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Name != "/app/config/key" {
		t.Errorf("Name = %q, want %q", result.Name, "/app/config/key")
	}
	if result.Value != "my-value" {
		t.Errorf("Value = %q, want %q", result.Value, "my-value")
	}
	if result.Type != "SecureString" {
		t.Errorf("Type = %q, want %q", result.Type, "SecureString")
	}
	if result.Description != "test desc" {
		t.Errorf("Description = %q, want %q", result.Description, "test desc")
	}
}

func TestSSMCreateModel_View_ContainsFields(t *testing.T) {
	m := NewSSMCreate("Create SSM Parameter", SSMCreateResult{})

	view := m.View()

	expectedStrings := []string{
		"Create SSM Parameter",
		"Parameter name",
		"Parameter value",
		"Parameter type",
		"Description",
		"String",
		"StringList",
		"SecureString",
		"submit",
		"cancel",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(view, s) {
			t.Errorf("View() should contain %q", s)
		}
	}
}

func TestSSMCreateModel_View_ShowsValidationError(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})
	m.err = "parameter name is required"

	view := m.View()
	if !strings.Contains(view, "parameter name is required") {
		t.Error("View() should show validation error")
	}
}

func TestSSMCreateModel_SubmittedAndCancelled(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})

	if m.Submitted() {
		t.Error("Submitted() should be false initially")
	}
	if m.Cancelled() {
		t.Error("Cancelled() should be false initially")
	}

	m.done = true
	if !m.Submitted() {
		t.Error("Submitted() should be true when done")
	}

	m.done = false
	m.quit = true
	if !m.Cancelled() {
		t.Error("Cancelled() should be true when quit")
	}
}

func TestSSMCreateModel_FocusWrapsAround(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})

	// Navigate to last field.
	m.focused = ssmCreateField(int(ssmFieldCount) - 1)

	// Tab should wrap to first field.
	m.focusNext()
	if m.focused != ssmFieldName {
		t.Errorf("after wrapping tab: focused = %d, want %d", m.focused, ssmFieldName)
	}

	// Shift+tab from first should wrap to last.
	m.focusPrev()
	if m.focused != ssmCreateField(int(ssmFieldCount)-1) {
		t.Errorf("after wrapping shift+tab: focused = %d, want %d", m.focused, ssmCreateField(int(ssmFieldCount)-1))
	}
}

func TestSSMCreateModel_EnterOnTypeFieldCyclesType(t *testing.T) {
	m := NewSSMCreate("", SSMCreateResult{})
	m.focused = ssmFieldType
	initialType := m.typeIndex

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SSMCreateModel)

	// Enter on type should cycle, not navigate.
	if m.focused != ssmFieldType {
		t.Errorf("enter on type field should keep focus on type, got focused=%d", m.focused)
	}
	expectedType := (initialType + 1) % len(ssmParamTypes)
	if m.typeIndex != expectedType {
		t.Errorf("typeIndex = %d, want %d", m.typeIndex, expectedType)
	}
}
