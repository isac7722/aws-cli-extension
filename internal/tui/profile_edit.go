package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// profileEditField enumerates the fields in the profile edit form.
type profileEditField int

const (
	fieldProfileName profileEditField = iota
	fieldAccessKeyID
	fieldSecretAccessKey
	fieldRegion
	fieldOutputFormat
	fieldSSOStartURL
	fieldSSORegion
	fieldSSOAccountID
	fieldSSORoleName
	fieldCount // sentinel — total number of fields
)

// ProfileEditResult holds the values collected from the profile edit form.
type ProfileEditResult struct {
	ProfileName     string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	OutputFormat    string
	SSOStartURL     string
	SSORegion       string
	SSOAccountID    string
	SSORoleName     string
}

// ProfileEditModel is a Bubble Tea model for editing an AWS profile.
// It presents a multi-field form with tab/shift-tab navigation.
type ProfileEditModel struct {
	inputs  []textinput.Model
	labels  []string
	focused profileEditField
	header  string
	done    bool
	quit    bool
	help    HelpOverlayModel
}

// NewProfileEdit creates a new profile edit form.
// If editing an existing profile, pass its current values via initial.
// For a new profile, pass a zero-value ProfileEditResult.
func NewProfileEdit(header string, initial ProfileEditResult) ProfileEditModel {
	inputs := make([]textinput.Model, fieldCount)
	labels := []string{
		"Profile name",
		"AWS Access Key ID",
		"AWS Secret Access Key",
		"Default region",
		"Output format",
		"SSO start URL",
		"SSO region",
		"SSO account ID",
		"SSO role name",
	}

	placeholders := []string{
		"e.g., production",
		"AKIA...",
		"wJalr...",
		"e.g., us-east-1",
		"json, yaml, text, table",
		"https://my-sso-portal.awsapps.com/start",
		"e.g., us-east-1",
		"123456789012",
		"e.g., ReadOnlyAccess",
	}

	initialValues := []string{
		initial.ProfileName,
		initial.AccessKeyID,
		initial.SecretAccessKey,
		initial.Region,
		initial.OutputFormat,
		initial.SSOStartURL,
		initial.SSORegion,
		initial.SSOAccountID,
		initial.SSORoleName,
	}

	for i := range inputs {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.CharLimit = 256
		if i == int(fieldSecretAccessKey) {
			ti.EchoMode = textinput.EchoPassword
		}
		if initialValues[i] != "" {
			ti.SetValue(initialValues[i])
		}
		inputs[i] = ti
	}

	// Focus the first field.
	inputs[0].Focus()

	return ProfileEditModel{
		inputs:  inputs,
		labels:  labels,
		focused: fieldProfileName,
		header:  header,
		help: NewHelpOverlayFromBindings("Profile Edit Keys",
			ProfileEditKeys.Next,
			ProfileEditKeys.Prev,
			ProfileEditKeys.Submit,
			ProfileEditKeys.Cancel,
			ProfileEditKeys.Help,
		),
	}
}

// Init implements tea.Model.
func (m ProfileEditModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m ProfileEditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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

		case "enter":
			if m.focused == profileEditField(int(fieldCount)-1) {
				// Submit on enter at last field.
				m.done = true
				return m, tea.Quit
			}
			m.focusNext()
			return m, textinput.Blink
		}
	}

	// Update the focused input.
	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m ProfileEditModel) View() string {
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
		fmt.Fprintf(&sb, "    %s\n", m.inputs[i].View())

		// Add a separator between credential fields and SSO fields.
		if i == int(fieldOutputFormat) {
			sb.WriteString("\n" + Dim.Render("  ── SSO Settings (optional) ──") + "\n\n")
		}
	}

	// Show help overlay if open, otherwise show compact help bar.
	if overlay := m.help.View(); overlay != "" {
		sb.WriteString("\n" + overlay)
	} else {
		sb.WriteString("\n" + HelpBar(
			ProfileEditKeys.Next,
			ProfileEditKeys.Prev,
			ProfileEditKeys.Submit,
			ProfileEditKeys.Cancel,
			ProfileEditKeys.Help,
		))
	}

	return sb.String()
}

// focusNext moves focus to the next field.
func (m *ProfileEditModel) focusNext() {
	m.inputs[m.focused].Blur()
	m.focused = (m.focused + 1) % profileEditField(fieldCount)
	m.inputs[m.focused].Focus()
}

// focusPrev moves focus to the previous field.
func (m *ProfileEditModel) focusPrev() {
	m.inputs[m.focused].Blur()
	m.focused--
	if m.focused < 0 {
		m.focused = profileEditField(int(fieldCount) - 1)
	}
	m.inputs[m.focused].Focus()
}

// Result returns the form values. Returns nil if the user cancelled.
func (m ProfileEditModel) Result() *ProfileEditResult {
	if m.quit {
		return nil
	}
	return &ProfileEditResult{
		ProfileName:     strings.TrimSpace(m.inputs[fieldProfileName].Value()),
		AccessKeyID:     strings.TrimSpace(m.inputs[fieldAccessKeyID].Value()),
		SecretAccessKey: strings.TrimSpace(m.inputs[fieldSecretAccessKey].Value()),
		Region:          strings.TrimSpace(m.inputs[fieldRegion].Value()),
		OutputFormat:    strings.TrimSpace(m.inputs[fieldOutputFormat].Value()),
		SSOStartURL:     strings.TrimSpace(m.inputs[fieldSSOStartURL].Value()),
		SSORegion:       strings.TrimSpace(m.inputs[fieldSSORegion].Value()),
		SSOAccountID:    strings.TrimSpace(m.inputs[fieldSSOAccountID].Value()),
		SSORoleName:     strings.TrimSpace(m.inputs[fieldSSORoleName].Value()),
	}
}

// Cancelled returns true if the user cancelled the form.
func (m ProfileEditModel) Cancelled() bool {
	return m.quit
}

// Submitted returns true if the user submitted the form.
func (m ProfileEditModel) Submitted() bool {
	return m.done
}

// RunProfileEdit runs the profile edit form and returns the result.
// Returns nil if the user cancelled.
func RunProfileEdit(header string, initial ProfileEditResult) (*ProfileEditResult, error) {
	m := NewProfileEdit(header, initial)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	return finalModel.(ProfileEditModel).Result(), nil
}
