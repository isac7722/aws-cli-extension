package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SSMBrowserOptions configures the SSM browser TUI.
type SSMBrowserOptions struct {
	Prefix  string
	Profile string
	Region  string
}

// SSMParam represents a single SSM parameter for display.
type SSMParam struct {
	Name  string
	Type  string // String, StringList, SecureString
	Value string
}

// ssmFetchedMsg is sent when parameters have been fetched.
type ssmFetchedMsg struct {
	params []SSMParam
	err    error
}

// SSMBrowserModel is the Bubble Tea model for browsing SSM parameters.
type SSMBrowserModel struct {
	options       SSMBrowserOptions
	params        []SSMParam
	cursor        int
	prefix        string
	loading       bool
	err           error
	quit          bool
	selectedValue string
	showValue     bool // whether to reveal the selected parameter's value
	help          HelpOverlayModel
}

// NewSSMBrowser creates a new SSM browser model.
func NewSSMBrowser(opts SSMBrowserOptions) SSMBrowserModel {
	if opts.Prefix == "" {
		opts.Prefix = "/"
	}
	return SSMBrowserModel{
		options: opts,
		prefix:  opts.Prefix,
		loading: true,
		help: NewHelpOverlayFromBindings("SSM Browser Keys",
			SSMBrowserKeys.Move,
			SSMBrowserKeys.ShowValue,
			SSMBrowserKeys.Select,
			SSMBrowserKeys.Cancel,
			SSMBrowserKeys.Help,
		),
	}
}

func (m SSMBrowserModel) Init() tea.Cmd {
	return m.fetchParams
}

func (m SSMBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ssmFetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.params = msg.params
		m.cursor = 0
		m.showValue = false
		return m, nil

	case tea.KeyMsg:
		// If there's an error, any key quits
		if m.err != nil {
			m.quit = true
			return m, tea.Quit
		}

		// Let help overlay handle '?' and esc-while-open first.
		if m.help.Update(msg) {
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.showValue = false
			}
		case "down", "j":
			if m.cursor < len(m.params)-1 {
				m.cursor++
				m.showValue = false
			}
		case "enter":
			if len(m.params) > 0 {
				m.selectedValue = m.params[m.cursor].Value
				return m, tea.Quit
			}
		case "v":
			// Toggle value visibility for current parameter
			if len(m.params) > 0 {
				m.showValue = !m.showValue
			}
		case "esc", "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SSMBrowserModel) View() string {
	var sb strings.Builder

	header := fmt.Sprintf("SSM Parameter Store — %s", m.prefix)
	sb.WriteString(Dim.Render(header) + "\n\n")

	if m.loading {
		sb.WriteString(Dim.Render("Loading parameters...") + "\n")
		return sb.String()
	}

	if m.err != nil {
		sb.WriteString(Red.Render("Error: "+m.err.Error()) + "\n\n")
		sb.WriteString(Dim.Render("Press any key to exit"))
		return sb.String()
	}

	if len(m.params) == 0 {
		sb.WriteString(Yellow.Render("No parameters found under "+m.prefix) + "\n\n")
		sb.WriteString(Dim.Render("esc/q: back"))
		return sb.String()
	}

	for i, param := range m.params {
		cursor := "  "
		if i == m.cursor {
			cursor = Cursor.Render("❯ ")
		}

		name := param.Name
		if i == m.cursor {
			name = Selected.Render(name)
		}

		typeHint := Dim.Render(fmt.Sprintf(" [%s]", param.Type))

		valuePart := ""
		if i == m.cursor && m.showValue {
			if param.Type == "SecureString" {
				valuePart = " " + Yellow.Render(param.Value)
			} else {
				valuePart = " " + Cyan.Render(param.Value)
			}
		} else if param.Type == "SecureString" {
			valuePart = " " + Dim.Render("••••••••")
		}

		fmt.Fprintf(&sb, "%s%s%s%s\n", cursor, name, typeHint, valuePart)
	}

	// Show help overlay if open, otherwise show compact help bar.
	if overlay := m.help.View(); overlay != "" {
		sb.WriteString("\n" + overlay)
	} else {
		sb.WriteString("\n" + HelpBar(
			SSMBrowserKeys.Move,
			SSMBrowserKeys.ShowValue,
			SSMBrowserKeys.Select,
			SSMBrowserKeys.Cancel,
			SSMBrowserKeys.Help,
		))
	}

	return sb.String()
}

// SelectedValue returns the value of the chosen parameter, or "" if cancelled.
func (m SSMBrowserModel) SelectedValue() string {
	if m.quit {
		return ""
	}
	return m.selectedValue
}

// fetchParams is a tea.Cmd placeholder that will be replaced with actual AWS SDK calls.
func (m SSMBrowserModel) fetchParams() tea.Msg {
	// TODO: Implement actual AWS SSM API call using AWS SDK for Go v2.
	// For now, return an empty list to allow the TUI to function.
	return ssmFetchedMsg{
		params: nil,
		err:    fmt.Errorf("SSM API not yet connected — use --profile and --region to configure"),
	}
}
