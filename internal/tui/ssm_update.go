package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ssmUpdateField enumerates the fields in the SSM parameter update form.
type ssmUpdateField int

const (
	ssmUpdateFieldValue ssmUpdateField = iota
	ssmUpdateFieldType                 // inline type selector
	ssmUpdateFieldDescription
	ssmUpdateFieldOverwrite // toggle
	ssmUpdateFieldConfirm   // version confirmation prompt
	ssmUpdateFieldCount     // sentinel — total number of fields
)

// SSMUpdateInput holds the existing parameter data used to pre-populate the update form.
type SSMUpdateInput struct {
	// Name is the parameter path (read-only, displayed but not editable).
	Name string
	// CurrentValue is the existing parameter value (displayed for reference).
	// For SecureString, this should be masked or empty if not decrypted.
	CurrentValue string
	// Type is the existing parameter type (String, StringList, SecureString).
	Type string
	// Description is the existing description.
	Description string
	// Version is the current version of the parameter.
	Version int64
	// IsSecureString indicates the parameter is a SecureString (for display masking).
	IsSecureString bool
}

// SSMUpdateResult holds the values collected from the SSM update form.
type SSMUpdateResult struct {
	Name        string
	Value       string
	Type        string
	Description string
	Overwrite   bool
}

// SSMUpdateModel is a Bubble Tea model for updating an existing SSM parameter.
// It displays the current parameter state (name, current value, version) as read-only
// context, and provides editable fields for the new value, type, description,
// an overwrite toggle, and a version confirmation prompt.
type SSMUpdateModel struct {
	input     SSMUpdateInput
	inputs    []textinput.Model
	labels    []string
	focused   ssmUpdateField
	typeIndex int  // index into ssmParamTypes for the type selector
	overwrite bool // overwrite toggle state
	confirmed bool // user confirmed the version update
	header    string
	done      bool
	quit      bool
	err       string // validation error message
}

// NewSSMUpdate creates a new SSM parameter update form pre-populated with the
// existing parameter values. The name and current value are displayed as read-only
// context; value, type, description, overwrite toggle, and version confirmation
// are editable.
func NewSSMUpdate(header string, input SSMUpdateInput) SSMUpdateModel {
	inputs := make([]textinput.Model, ssmUpdateFieldCount)
	labels := []string{
		"New value",
		"Parameter type",
		"Description",
		"Overwrite existing",
		"Confirm update",
	}

	// Value field — pre-populated with current value (unless SecureString).
	valueInput := textinput.New()
	valueInput.Placeholder = "new parameter value"
	valueInput.CharLimit = 4096
	if !input.IsSecureString && input.CurrentValue != "" {
		valueInput.SetValue(input.CurrentValue)
	}
	inputs[ssmUpdateFieldValue] = valueInput

	// Type field — placeholder only (type uses inline selector).
	typeInput := textinput.New()
	inputs[ssmUpdateFieldType] = typeInput

	// Description field.
	descInput := textinput.New()
	descInput.Placeholder = "optional description"
	descInput.CharLimit = 1024
	if input.Description != "" {
		descInput.SetValue(input.Description)
	}
	inputs[ssmUpdateFieldDescription] = descInput

	// Overwrite and Confirm fields use toggles, not text inputs.
	inputs[ssmUpdateFieldOverwrite] = textinput.New()
	inputs[ssmUpdateFieldConfirm] = textinput.New()

	// Focus the first editable field.
	inputs[ssmUpdateFieldValue].Focus()

	// Determine initial type index from existing parameter type.
	typeIndex := 0
	if input.Type != "" {
		for i, t := range ssmParamTypes {
			if t == input.Type {
				typeIndex = i
				break
			}
		}
	}

	return SSMUpdateModel{
		input:     input,
		inputs:    inputs,
		labels:    labels,
		focused:   ssmUpdateFieldValue,
		typeIndex: typeIndex,
		overwrite: true, // default to true for updates
		confirmed: false,
		header:    header,
	}
}

// Init implements tea.Model.
func (m SSMUpdateModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m SSMUpdateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Clear any validation error on new input.
		m.err = ""

		switch msg.String() {
		case "ctrl+c", "esc":
			m.quit = true
			return m, tea.Quit

		case "tab", "down":
			m.focusNext()
			return m, textinput.Blink

		case "shift+tab", "up":
			m.focusPrev()
			return m, textinput.Blink

		case "left":
			if m.focused == ssmUpdateFieldType {
				m.typePrev()
				return m, nil
			}

		case "right":
			if m.focused == ssmUpdateFieldType {
				m.typeNext()
				return m, nil
			}

		case " ", "enter":
			// Toggle fields respond to space and enter.
			if m.focused == ssmUpdateFieldOverwrite {
				m.overwrite = !m.overwrite
				return m, nil
			}
			if m.focused == ssmUpdateFieldConfirm {
				m.confirmed = !m.confirmed
				return m, nil
			}

			// Type field: enter/space cycles type forward.
			if m.focused == ssmUpdateFieldType {
				m.typeNext()
				return m, nil
			}

			// Enter on text fields: navigate to next or submit on last text field.
			if msg.String() == "enter" {
				m.focusNext()
				return m, textinput.Blink
			}

		case "ctrl+s":
			// Submit from any field.
			if err := m.validate(); err != "" {
				m.err = err
				return m, nil
			}
			m.done = true
			return m, tea.Quit
		}
	}

	// Update the focused text input (only for value and description fields).
	if m.focused == ssmUpdateFieldValue || m.focused == ssmUpdateFieldDescription {
		var cmd tea.Cmd
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model.
func (m SSMUpdateModel) View() string {
	var sb strings.Builder

	if m.header != "" {
		sb.WriteString(Bold.Render(m.header) + "\n\n")
	}

	// Read-only context: parameter name and current value.
	sb.WriteString(Dim.Render("  Parameter: ") + Cyan.Render(m.input.Name) + "\n")
	sb.WriteString(Dim.Render("  Version:   ") + fmt.Sprintf("%d", m.input.Version) + "\n")

	// Current value display — mask SecureString values.
	currentDisplay := m.input.CurrentValue
	if m.input.IsSecureString {
		if currentDisplay == "" {
			currentDisplay = Yellow.Render("(SecureString — not decrypted)")
		} else {
			currentDisplay = Yellow.Render("****")
		}
	} else if currentDisplay == "" {
		currentDisplay = Dim.Render("(empty)")
	}
	sb.WriteString(Dim.Render("  Current:   ") + currentDisplay + "\n")
	sb.WriteString("\n")

	// Editable fields.
	for i, label := range m.labels {
		field := ssmUpdateField(i)
		cursor := "  "
		var styledLabel string
		if field == m.focused {
			cursor = Cursor.Render("❯ ")
			styledLabel = Selected.Render(label)
		} else {
			styledLabel = Dim.Render(label)
		}

		fmt.Fprintf(&sb, "%s%s\n", cursor, styledLabel)

		switch field {
		case ssmUpdateFieldValue:
			fmt.Fprintf(&sb, "    %s\n", m.inputs[ssmUpdateFieldValue].View())

		case ssmUpdateFieldType:
			sb.WriteString("    ")
			sb.WriteString(m.renderTypeSelector())
			sb.WriteString("\n")

		case ssmUpdateFieldDescription:
			fmt.Fprintf(&sb, "    %s\n", m.inputs[ssmUpdateFieldDescription].View())

		case ssmUpdateFieldOverwrite:
			sb.WriteString("    ")
			sb.WriteString(m.renderToggle(m.overwrite))
			sb.WriteString("\n")

		case ssmUpdateFieldConfirm:
			confirmMsg := fmt.Sprintf("Update version %d → %d", m.input.Version, m.input.Version+1)
			sb.WriteString("    ")
			sb.WriteString(m.renderToggle(m.confirmed))
			sb.WriteString("  ")
			sb.WriteString(Dim.Render(confirmMsg))
			sb.WriteString("\n")
		}
	}

	if m.err != "" {
		sb.WriteString("\n" + Red.Render("  ✘ "+m.err))
	}

	sb.WriteString("\n" + Dim.Render("tab/↓: next  shift+tab/↑: prev  ←→: change type  space: toggle  ctrl+s: submit  esc: cancel"))

	return sb.String()
}

// renderTypeSelector renders the inline type selector with highlighted current type.
func (m SSMUpdateModel) renderTypeSelector() string {
	var parts []string
	for i, t := range ssmParamTypes {
		if i == m.typeIndex {
			if t == "SecureString" {
				parts = append(parts, Yellow.Render("▸ "+t))
			} else {
				parts = append(parts, Selected.Render("▸ "+t))
			}
		} else {
			parts = append(parts, Dim.Render("  "+t))
		}
	}
	return strings.Join(parts, "  ")
}

// renderToggle renders a checkbox-style toggle.
func (m SSMUpdateModel) renderToggle(on bool) string {
	if on {
		return Green.Render("[✔]")
	}
	return Dim.Render("[ ]")
}

// validate checks required fields and returns an error message, or empty string if valid.
func (m SSMUpdateModel) validate() string {
	value := strings.TrimSpace(m.inputs[ssmUpdateFieldValue].Value())
	if value == "" {
		return "new parameter value is required"
	}

	if !m.confirmed {
		return "please confirm the version update before submitting"
	}

	return ""
}

// focusNext moves focus to the next field.
func (m *SSMUpdateModel) focusNext() {
	m.blurCurrent()
	m.focused = (m.focused + 1) % ssmUpdateField(ssmUpdateFieldCount)
	m.focusCurrent()
}

// focusPrev moves focus to the previous field.
func (m *SSMUpdateModel) focusPrev() {
	m.blurCurrent()
	m.focused--
	if m.focused < 0 {
		m.focused = ssmUpdateField(int(ssmUpdateFieldCount) - 1)
	}
	m.focusCurrent()
}

// blurCurrent blurs the currently focused text input (if applicable).
func (m *SSMUpdateModel) blurCurrent() {
	if m.focused == ssmUpdateFieldValue || m.focused == ssmUpdateFieldDescription {
		m.inputs[m.focused].Blur()
	}
}

// focusCurrent focuses the current text input (if applicable).
func (m *SSMUpdateModel) focusCurrent() {
	if m.focused == ssmUpdateFieldValue || m.focused == ssmUpdateFieldDescription {
		m.inputs[m.focused].Focus()
	}
}

// typeNext cycles the type selector forward.
func (m *SSMUpdateModel) typeNext() {
	m.typeIndex = (m.typeIndex + 1) % len(ssmParamTypes)
}

// typePrev cycles the type selector backward.
func (m *SSMUpdateModel) typePrev() {
	m.typeIndex--
	if m.typeIndex < 0 {
		m.typeIndex = len(ssmParamTypes) - 1
	}
}

// Result returns the form values. Returns nil if the user cancelled.
func (m SSMUpdateModel) Result() *SSMUpdateResult {
	if m.quit {
		return nil
	}
	return &SSMUpdateResult{
		Name:        m.input.Name,
		Value:       strings.TrimSpace(m.inputs[ssmUpdateFieldValue].Value()),
		Type:        ssmParamTypes[m.typeIndex],
		Description: strings.TrimSpace(m.inputs[ssmUpdateFieldDescription].Value()),
		Overwrite:   m.overwrite,
	}
}

// Cancelled returns true if the user cancelled the form.
func (m SSMUpdateModel) Cancelled() bool {
	return m.quit
}

// Submitted returns true if the user submitted the form.
func (m SSMUpdateModel) Submitted() bool {
	return m.done
}

// RunSSMUpdate runs the SSM parameter update form and returns the result.
// Returns nil if the user cancelled.
func RunSSMUpdate(header string, input SSMUpdateInput) (*SSMUpdateResult, error) {
	m := NewSSMUpdate(header, input)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	return finalModel.(SSMUpdateModel).Result(), nil
}
