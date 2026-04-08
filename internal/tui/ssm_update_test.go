package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSSMUpdate_DefaultValues(t *testing.T) {
	input := SSMUpdateInput{
		Name:         "/app/config/db_host",
		CurrentValue: "old-host.example.com",
		Type:         "String",
		Version:      3,
	}
	m := NewSSMUpdate("Update Parameter", input)

	if m.header != "Update Parameter" {
		t.Errorf("header = %q, want %q", m.header, "Update Parameter")
	}
	if m.focused != ssmUpdateFieldValue {
		t.Errorf("focused = %d, want %d (ssmUpdateFieldValue)", m.focused, ssmUpdateFieldValue)
	}
	if m.typeIndex != 0 {
		t.Errorf("typeIndex = %d, want 0 (String)", m.typeIndex)
	}
	if !m.overwrite {
		t.Error("overwrite should default to true for updates")
	}
	if m.confirmed {
		t.Error("confirmed should default to false")
	}
	if m.done || m.quit {
		t.Error("form should not be done or quit on creation")
	}
}

func TestNewSSMUpdate_PrePopulatesValue(t *testing.T) {
	input := SSMUpdateInput{
		Name:         "/app/config/db_host",
		CurrentValue: "old-host.example.com",
		Type:         "String",
		Description:  "Database host",
		Version:      5,
	}
	m := NewSSMUpdate("", input)

	// Value field should be pre-populated with current value.
	if m.inputs[ssmUpdateFieldValue].Value() != "old-host.example.com" {
		t.Errorf("value = %q, want %q", m.inputs[ssmUpdateFieldValue].Value(), "old-host.example.com")
	}

	// Description should be pre-populated.
	if m.inputs[ssmUpdateFieldDescription].Value() != "Database host" {
		t.Errorf("description = %q, want %q", m.inputs[ssmUpdateFieldDescription].Value(), "Database host")
	}
}

func TestNewSSMUpdate_SecureStringNotPrePopulated(t *testing.T) {
	input := SSMUpdateInput{
		Name:           "/app/config/db_pass",
		CurrentValue:   "secret-password",
		Type:           "SecureString",
		Version:        2,
		IsSecureString: true,
	}
	m := NewSSMUpdate("", input)

	// Value field should NOT be pre-populated for SecureString.
	if m.inputs[ssmUpdateFieldValue].Value() != "" {
		t.Errorf("SecureString value should not be pre-populated, got %q", m.inputs[ssmUpdateFieldValue].Value())
	}

	// Type index should be 2 (SecureString).
	if m.typeIndex != 2 {
		t.Errorf("typeIndex = %d, want 2 (SecureString)", m.typeIndex)
	}
}

func TestSSMUpdateModel_FocusNavigation(t *testing.T) {
	input := SSMUpdateInput{
		Name:    "/app/test",
		Type:    "String",
		Version: 1,
	}
	m := NewSSMUpdate("", input)

	// Initial focus on value field.
	if m.focused != ssmUpdateFieldValue {
		t.Fatalf("initial focused = %d, want %d", m.focused, ssmUpdateFieldValue)
	}

	// Tab moves to type field.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(SSMUpdateModel)
	if m.focused != ssmUpdateFieldType {
		t.Errorf("after tab: focused = %d, want %d (type)", m.focused, ssmUpdateFieldType)
	}

	// Tab moves to description.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(SSMUpdateModel)
	if m.focused != ssmUpdateFieldDescription {
		t.Errorf("after 2nd tab: focused = %d, want %d (description)", m.focused, ssmUpdateFieldDescription)
	}

	// Tab moves to overwrite.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(SSMUpdateModel)
	if m.focused != ssmUpdateFieldOverwrite {
		t.Errorf("after 3rd tab: focused = %d, want %d (overwrite)", m.focused, ssmUpdateFieldOverwrite)
	}

	// Tab moves to confirm.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(SSMUpdateModel)
	if m.focused != ssmUpdateFieldConfirm {
		t.Errorf("after 4th tab: focused = %d, want %d (confirm)", m.focused, ssmUpdateFieldConfirm)
	}

	// Tab wraps back to value.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(SSMUpdateModel)
	if m.focused != ssmUpdateFieldValue {
		t.Errorf("after wrap tab: focused = %d, want %d (value)", m.focused, ssmUpdateFieldValue)
	}

	// Shift+tab wraps to confirm.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(SSMUpdateModel)
	if m.focused != ssmUpdateFieldConfirm {
		t.Errorf("after shift+tab: focused = %d, want %d (confirm)", m.focused, ssmUpdateFieldConfirm)
	}
}

func TestSSMUpdateModel_TypeCycling(t *testing.T) {
	input := SSMUpdateInput{
		Name:    "/app/test",
		Type:    "String",
		Version: 1,
	}
	m := NewSSMUpdate("", input)
	m.focused = ssmUpdateFieldType

	if ssmParamTypes[m.typeIndex] != "String" {
		t.Fatalf("initial type = %q, want %q", ssmParamTypes[m.typeIndex], "String")
	}

	// Right arrow cycles forward.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(SSMUpdateModel)
	if ssmParamTypes[m.typeIndex] != "StringList" {
		t.Errorf("after right: type = %q, want %q", ssmParamTypes[m.typeIndex], "StringList")
	}

	// Right again.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(SSMUpdateModel)
	if ssmParamTypes[m.typeIndex] != "SecureString" {
		t.Errorf("after 2nd right: type = %q, want %q", ssmParamTypes[m.typeIndex], "SecureString")
	}

	// Wraps around.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(SSMUpdateModel)
	if ssmParamTypes[m.typeIndex] != "String" {
		t.Errorf("after 3rd right: type = %q, want %q", ssmParamTypes[m.typeIndex], "String")
	}

	// Left wraps backward.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(SSMUpdateModel)
	if ssmParamTypes[m.typeIndex] != "SecureString" {
		t.Errorf("after left: type = %q, want %q", ssmParamTypes[m.typeIndex], "SecureString")
	}
}

func TestSSMUpdateModel_OverwriteToggle(t *testing.T) {
	input := SSMUpdateInput{
		Name:    "/app/test",
		Type:    "String",
		Version: 1,
	}
	m := NewSSMUpdate("", input)
	m.focused = ssmUpdateFieldOverwrite

	// Default is true.
	if !m.overwrite {
		t.Fatal("overwrite should default to true")
	}

	// Space toggles off.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(SSMUpdateModel)
	if m.overwrite {
		t.Error("overwrite should be false after space toggle")
	}

	// Space toggles back on.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(SSMUpdateModel)
	if !m.overwrite {
		t.Error("overwrite should be true after second space toggle")
	}

	// Enter also toggles.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SSMUpdateModel)
	if m.overwrite {
		t.Error("overwrite should be false after enter toggle")
	}
}

func TestSSMUpdateModel_ConfirmToggle(t *testing.T) {
	input := SSMUpdateInput{
		Name:    "/app/test",
		Type:    "String",
		Version: 1,
	}
	m := NewSSMUpdate("", input)
	m.focused = ssmUpdateFieldConfirm

	// Default is false.
	if m.confirmed {
		t.Fatal("confirmed should default to false")
	}

	// Space toggles on.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(SSMUpdateModel)
	if !m.confirmed {
		t.Error("confirmed should be true after space toggle")
	}

	// Space toggles back off.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(SSMUpdateModel)
	if m.confirmed {
		t.Error("confirmed should be false after second space toggle")
	}
}

func TestSSMUpdateModel_Cancel(t *testing.T) {
	input := SSMUpdateInput{Name: "/app/test", Type: "String", Version: 1}
	m := NewSSMUpdate("", input)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(SSMUpdateModel)

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

func TestSSMUpdateModel_CtrlCCancel(t *testing.T) {
	input := SSMUpdateInput{Name: "/app/test", Type: "String", Version: 1}
	m := NewSSMUpdate("", input)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(SSMUpdateModel)

	if !m.Cancelled() {
		t.Error("expected Cancelled() to return true on ctrl+c")
	}
}

func TestSSMUpdateModel_Validate_EmptyValue(t *testing.T) {
	input := SSMUpdateInput{Name: "/app/test", Type: "String", Version: 1}
	m := NewSSMUpdate("", input)
	m.confirmed = true

	err := m.validate()
	if err == "" {
		t.Error("expected validation error for empty value")
	}
	if !strings.Contains(err, "value is required") {
		t.Errorf("error = %q, want it to contain 'value is required'", err)
	}
}

func TestSSMUpdateModel_Validate_NotConfirmed(t *testing.T) {
	input := SSMUpdateInput{Name: "/app/test", Type: "String", Version: 1}
	m := NewSSMUpdate("", input)
	m.inputs[ssmUpdateFieldValue].SetValue("new-value")
	// confirmed is false by default

	err := m.validate()
	if err == "" {
		t.Error("expected validation error for unconfirmed update")
	}
	if !strings.Contains(err, "confirm") {
		t.Errorf("error = %q, want it to contain 'confirm'", err)
	}
}

func TestSSMUpdateModel_Validate_Success(t *testing.T) {
	input := SSMUpdateInput{Name: "/app/test", Type: "String", Version: 1}
	m := NewSSMUpdate("", input)
	m.inputs[ssmUpdateFieldValue].SetValue("new-value")
	m.confirmed = true

	err := m.validate()
	if err != "" {
		t.Errorf("expected no validation error, got %q", err)
	}
}

func TestSSMUpdateModel_Result(t *testing.T) {
	input := SSMUpdateInput{
		Name:         "/app/config/key",
		CurrentValue: "old-value",
		Type:         "String",
		Description:  "old desc",
		Version:      3,
	}
	m := NewSSMUpdate("", input)
	m.inputs[ssmUpdateFieldValue].SetValue("new-value")
	m.inputs[ssmUpdateFieldDescription].SetValue("new desc")
	m.typeIndex = 1 // StringList
	m.overwrite = true
	m.done = true

	result := m.Result()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Name != "/app/config/key" {
		t.Errorf("Name = %q, want %q", result.Name, "/app/config/key")
	}
	if result.Value != "new-value" {
		t.Errorf("Value = %q, want %q", result.Value, "new-value")
	}
	if result.Type != "StringList" {
		t.Errorf("Type = %q, want %q", result.Type, "StringList")
	}
	if result.Description != "new desc" {
		t.Errorf("Description = %q, want %q", result.Description, "new desc")
	}
	if !result.Overwrite {
		t.Error("Overwrite should be true")
	}
}

func TestSSMUpdateModel_View_ContainsFields(t *testing.T) {
	input := SSMUpdateInput{
		Name:         "/app/config/db_host",
		CurrentValue: "old-host",
		Type:         "String",
		Version:      3,
	}
	m := NewSSMUpdate("Update SSM Parameter", input)

	view := m.View()

	expectedStrings := []string{
		"Update SSM Parameter",
		"/app/config/db_host",
		"New value",
		"Parameter type",
		"Description",
		"Overwrite existing",
		"Confirm update",
		"String",
		"StringList",
		"SecureString",
		"submit",
		"cancel",
		"3",     // version display
		"3 → 4", // version confirmation message
	}

	for _, s := range expectedStrings {
		if !strings.Contains(view, s) {
			t.Errorf("View() should contain %q", s)
		}
	}
}

func TestSSMUpdateModel_View_MasksSecureString(t *testing.T) {
	input := SSMUpdateInput{
		Name:           "/app/config/secret",
		CurrentValue:   "super-secret",
		Type:           "SecureString",
		Version:        2,
		IsSecureString: true,
	}
	m := NewSSMUpdate("", input)

	view := m.View()

	// Should NOT show the actual secret value.
	if strings.Contains(view, "super-secret") {
		t.Error("View() should not display SecureString value in plain text")
	}

	// Should show masking indicator.
	if !strings.Contains(view, "****") {
		t.Error("View() should mask SecureString current value")
	}
}

func TestSSMUpdateModel_View_SecureStringNotDecrypted(t *testing.T) {
	input := SSMUpdateInput{
		Name:           "/app/config/secret",
		CurrentValue:   "",
		Type:           "SecureString",
		Version:        2,
		IsSecureString: true,
	}
	m := NewSSMUpdate("", input)

	view := m.View()

	if !strings.Contains(view, "not decrypted") {
		t.Error("View() should show not decrypted message for empty SecureString")
	}
}

func TestSSMUpdateModel_View_ShowsValidationError(t *testing.T) {
	input := SSMUpdateInput{Name: "/app/test", Type: "String", Version: 1}
	m := NewSSMUpdate("", input)
	m.err = "new parameter value is required"

	view := m.View()
	if !strings.Contains(view, "new parameter value is required") {
		t.Error("View() should show validation error")
	}
}

func TestSSMUpdateModel_View_ShowsToggleStates(t *testing.T) {
	input := SSMUpdateInput{Name: "/app/test", Type: "String", Version: 1}
	m := NewSSMUpdate("", input)

	// Overwrite on (default), confirmed off.
	view := m.View()
	// The view should contain both toggle indicators.
	if !strings.Contains(view, "Overwrite") {
		t.Error("View() should show Overwrite label")
	}
	if !strings.Contains(view, "Confirm") {
		t.Error("View() should show Confirm label")
	}
}

func TestSSMUpdateModel_SubmittedAndCancelled(t *testing.T) {
	input := SSMUpdateInput{Name: "/app/test", Type: "String", Version: 1}
	m := NewSSMUpdate("", input)

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

func TestSSMUpdateModel_FocusWrapsAround(t *testing.T) {
	input := SSMUpdateInput{Name: "/app/test", Type: "String", Version: 1}
	m := NewSSMUpdate("", input)

	// Navigate to last field.
	m.focused = ssmUpdateFieldConfirm

	// Tab should wrap to first field.
	m.focusNext()
	if m.focused != ssmUpdateFieldValue {
		t.Errorf("after wrap tab: focused = %d, want %d", m.focused, ssmUpdateFieldValue)
	}

	// Shift+tab from first should wrap to last.
	m.focusPrev()
	if m.focused != ssmUpdateFieldConfirm {
		t.Errorf("after wrap shift+tab: focused = %d, want %d", m.focused, ssmUpdateFieldConfirm)
	}
}

func TestSSMUpdateModel_EnterOnTypeFieldCyclesType(t *testing.T) {
	input := SSMUpdateInput{Name: "/app/test", Type: "String", Version: 1}
	m := NewSSMUpdate("", input)
	m.focused = ssmUpdateFieldType
	initialType := m.typeIndex

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SSMUpdateModel)

	// Enter on type should cycle, not navigate.
	if m.focused != ssmUpdateFieldType {
		t.Errorf("enter on type field should keep focus on type, got focused=%d", m.focused)
	}
	expectedType := (initialType + 1) % len(ssmParamTypes)
	if m.typeIndex != expectedType {
		t.Errorf("typeIndex = %d, want %d", m.typeIndex, expectedType)
	}
}

func TestSSMUpdateModel_CtrlS_SubmitWithValidation(t *testing.T) {
	input := SSMUpdateInput{Name: "/app/test", Type: "String", Version: 1}
	m := NewSSMUpdate("", input)
	m.inputs[ssmUpdateFieldValue].SetValue("new-value")
	m.confirmed = true

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(SSMUpdateModel)

	if !m.Submitted() {
		t.Error("expected Submitted() to be true after ctrl+s with valid input")
	}
	if cmd == nil {
		t.Error("expected quit command on successful submit")
	}
}

func TestSSMUpdateModel_CtrlS_FailsValidation(t *testing.T) {
	input := SSMUpdateInput{Name: "/app/test", Type: "String", Version: 1}
	m := NewSSMUpdate("", input)
	// No value set, not confirmed

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(SSMUpdateModel)

	if m.Submitted() {
		t.Error("should not submit with invalid input")
	}
	if m.err == "" {
		t.Error("expected validation error")
	}
}

func TestSSMUpdateModel_NameIsReadOnly(t *testing.T) {
	input := SSMUpdateInput{
		Name:    "/app/config/key",
		Type:    "String",
		Version: 1,
	}
	m := NewSSMUpdate("", input)

	// Name should come from input, not from any editable field.
	result := m.Result()
	if result.Name != "/app/config/key" {
		t.Errorf("Name = %q, want %q (should come from read-only input)", result.Name, "/app/config/key")
	}
}
