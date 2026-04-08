package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ssmCreateField enumerates the fields in the SSM parameter create form.
type ssmCreateField int

const (
	ssmFieldName ssmCreateField = iota
	ssmFieldValue
	ssmFieldType
	ssmFieldDescription
	ssmFieldCount // sentinel — total number of fields
)

// Valid SSM parameter types for the type selector.
var ssmParamTypes = []string{"String", "StringList", "SecureString"}

// SSMCreateResult holds the values collected from the SSM create form.
type SSMCreateResult struct {
	Name        string
	Value       string
	Type        string
	Description string
}

// SSMCreateModel is a Bubble Tea model for creating an SSM parameter.
// It presents a multi-field form with text inputs for name, value, and description,
// and an inline selector for parameter type.
type SSMCreateModel struct {
	inputs    []textinput.Model
	labels    []string
	focused   ssmCreateField
	typeIndex int // index into ssmParamTypes for the type selector
	header    string
	done      bool
	quit      bool
	err       string // validation error message
	help      HelpOverlayModel
}

// NewSSMCreate creates a new SSM parameter creation form.
// If editing defaults, pass initial values via initial; otherwise pass a zero-value.
func NewSSMCreate(header string, initial SSMCreateResult) SSMCreateModel {
	inputs := make([]textinput.Model, ssmFieldCount)
	labels := []string{
		"Parameter name",
		"Parameter value",
		"Parameter type",
		"Description",
	}

	placeholders := []string{
		"/app/config/my-param",
		"parameter value",
		"", // type uses inline selector, not text input
		"optional description",
	}

	initialValues := []string{
		initial.Name,
		initial.Value,
		"", // type handled separately
		initial.Description,
	}

	for i := range inputs {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.CharLimit = 2048
		if initialValues[i] != "" {
			ti.SetValue(initialValues[i])
		}
		inputs[i] = ti
	}

	// Focus the first field.
	inputs[0].Focus()

	// Determine initial type index.
	typeIndex := 0
	if initial.Type != "" {
		for i, t := range ssmParamTypes {
			if t == initial.Type {
				typeIndex = i
				break
			}
		}
	}

	return SSMCreateModel{
		inputs:    inputs,
		labels:    labels,
		focused:   ssmFieldName,
		typeIndex: typeIndex,
		header:    header,
		help: NewHelpOverlayFromBindings("SSM Create Keys",
			SSMCreateKeys.Next,
			SSMCreateKeys.Prev,
			SSMCreateKeys.ChangeType,
			SSMCreateKeys.Submit,
			SSMCreateKeys.Cancel,
			SSMCreateKeys.Help,
		),
	}
}

// Init implements tea.Model.
func (m SSMCreateModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m SSMCreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Clear any validation error on new input.
		m.err = ""

		// Let help overlay handle '?' and esc-while-open first.
		if m.help.Update(msg) {
			return m, nil
		}

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
			if m.focused == ssmFieldType {
				m.typePrev()
				return m, nil
			}

		case "right":
			if m.focused == ssmFieldType {
				m.typeNext()
				return m, nil
			}

		case "enter":
			if m.focused == ssmFieldType {
				// On the type field, enter cycles type forward.
				m.typeNext()
				return m, nil
			}
			if m.focused == ssmCreateField(int(ssmFieldCount)-1) {
				// Submit on enter at last field.
				if err := m.validate(); err != "" {
					m.err = err
					return m, nil
				}
				m.done = true
				return m, tea.Quit
			}
			m.focusNext()
			return m, textinput.Blink

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

	// Update the focused input (skip type field which uses selector).
	if m.focused != ssmFieldType {
		var cmd tea.Cmd
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model.
func (m SSMCreateModel) View() string {
	var sb strings.Builder

	if m.header != "" {
		sb.WriteString(Bold.Render(m.header) + "\n\n")
	}

	for i, label := range m.labels {
		cursor := "  "
		if i == int(m.focused) {
			cursor = Cursor.Render("❯ ")
			label = Selected.Render(label)
		} else {
			label = Dim.Render(label)
		}

		fmt.Fprintf(&sb, "%s%s\n", cursor, label)

		if ssmCreateField(i) == ssmFieldType {
			// Render type selector inline.
			sb.WriteString("    ")
			sb.WriteString(m.renderTypeSelector())
			sb.WriteString("\n")
		} else {
			fmt.Fprintf(&sb, "    %s\n", m.inputs[i].View())
		}
	}

	if m.err != "" {
		sb.WriteString("\n" + Red.Render("  ✘ "+m.err))
	}

	// Show help overlay if open, otherwise show compact help bar.
	if overlay := m.help.View(); overlay != "" {
		sb.WriteString("\n" + overlay)
	} else {
		sb.WriteString("\n" + HelpBar(
			SSMCreateKeys.Next,
			SSMCreateKeys.Prev,
			SSMCreateKeys.ChangeType,
			SSMCreateKeys.Submit,
			SSMCreateKeys.Cancel,
			SSMCreateKeys.Help,
		))
	}

	return sb.String()
}

// renderTypeSelector renders the inline type selector with highlighted current type.
func (m SSMCreateModel) renderTypeSelector() string {
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

// validate checks required fields and returns an error message, or empty string if valid.
func (m SSMCreateModel) validate() string {
	name := strings.TrimSpace(m.inputs[ssmFieldName].Value())
	if name == "" {
		return "parameter name is required"
	}
	if !strings.HasPrefix(name, "/") {
		return "parameter name must start with '/'"
	}

	value := strings.TrimSpace(m.inputs[ssmFieldValue].Value())
	if value == "" {
		return "parameter value is required"
	}

	return ""
}

// focusNext moves focus to the next field.
func (m *SSMCreateModel) focusNext() {
	m.inputs[m.focused].Blur()
	m.focused = (m.focused + 1) % ssmCreateField(ssmFieldCount)
	if m.focused != ssmFieldType {
		m.inputs[m.focused].Focus()
	}
}

// focusPrev moves focus to the previous field.
func (m *SSMCreateModel) focusPrev() {
	m.inputs[m.focused].Blur()
	m.focused--
	if m.focused < 0 {
		m.focused = ssmCreateField(int(ssmFieldCount) - 1)
	}
	if m.focused != ssmFieldType {
		m.inputs[m.focused].Focus()
	}
}

// typeNext cycles the type selector forward.
func (m *SSMCreateModel) typeNext() {
	m.typeIndex = (m.typeIndex + 1) % len(ssmParamTypes)
}

// typePrev cycles the type selector backward.
func (m *SSMCreateModel) typePrev() {
	m.typeIndex--
	if m.typeIndex < 0 {
		m.typeIndex = len(ssmParamTypes) - 1
	}
}

// Result returns the form values. Returns nil if the user cancelled.
func (m SSMCreateModel) Result() *SSMCreateResult {
	if m.quit {
		return nil
	}
	return &SSMCreateResult{
		Name:        strings.TrimSpace(m.inputs[ssmFieldName].Value()),
		Value:       strings.TrimSpace(m.inputs[ssmFieldValue].Value()),
		Type:        ssmParamTypes[m.typeIndex],
		Description: strings.TrimSpace(m.inputs[ssmFieldDescription].Value()),
	}
}

// Cancelled returns true if the user cancelled the form.
func (m SSMCreateModel) Cancelled() bool {
	return m.quit
}

// Submitted returns true if the user submitted the form.
func (m SSMCreateModel) Submitted() bool {
	return m.done
}

// RunSSMCreate runs the SSM parameter creation form and returns the result.
// Returns nil if the user cancelled.
func RunSSMCreate(header string, initial SSMCreateResult) (*SSMCreateResult, error) {
	m := NewSSMCreate(header, initial)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	return finalModel.(SSMCreateModel).Result(), nil
}
