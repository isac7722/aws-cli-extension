package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmStyle controls the visual appearance of the confirmation prompt.
type ConfirmStyle int

const (
	// ConfirmStyleDefault is a neutral confirmation prompt (cyan icon).
	ConfirmStyleDefault ConfirmStyle = iota
	// ConfirmStyleDestructive is a destructive confirmation prompt (red icon/highlight).
	ConfirmStyleDestructive
)

// ConfirmModel is a Y/N confirmation prompt with optional item listing
// for single and batch delete operations.
type ConfirmModel struct {
	message string
	items   []string     // items being affected (shown in a list above the prompt)
	style   ConfirmStyle // visual style (default vs destructive)
	confirm bool
	done    bool
	focused bool // whether 'y' or 'n' side is focused (left/right arrow toggle)
	help    HelpOverlayModel
}

// ConfirmOption is a functional option for NewConfirm.
type ConfirmOption func(*ConfirmModel)

// WithItems adds a list of items to display above the prompt.
func WithItems(items []string) ConfirmOption {
	return func(m *ConfirmModel) {
		m.items = items
	}
}

// WithDestructive sets the confirmation style to destructive (red).
func WithDestructive() ConfirmOption {
	return func(m *ConfirmModel) {
		m.style = ConfirmStyleDestructive
	}
}

// NewConfirm creates a new confirmation prompt model.
func NewConfirm(message string, opts ...ConfirmOption) ConfirmModel {
	m := ConfirmModel{
		message: message,
		help: NewHelpOverlayFromBindings("Confirm Keys",
			ConfirmKeys.Confirm,
			ConfirmKeys.Deny,
			ConfirmKeys.Toggle,
			ConfirmKeys.Cancel,
		),
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func (m ConfirmModel) Init() tea.Cmd { return nil }

func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Let help overlay handle '?' and esc-while-open first.
		if m.help.Update(msg) {
			return m, nil
		}

		switch strings.ToLower(msg.String()) {
		case "y":
			m.confirm = true
			m.done = true
			return m, tea.Quit
		case "n":
			m.confirm = false
			m.done = true
			return m, tea.Quit
		case "left", "right", "h", "l", "tab":
			m.focused = !m.focused
		case "enter":
			m.confirm = m.focused
			m.done = true
			return m, tea.Quit
		case "esc", "ctrl+c", "q":
			m.confirm = false
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ConfirmModel) View() string {
	var sb strings.Builder

	// Render items list if present.
	if len(m.items) > 0 {
		sb.WriteString("\n")
		if len(m.items) == 1 {
			fmt.Fprintf(&sb, "  %s %s\n", Dim.Render("Parameter:"), Bold.Render(m.items[0]))
		} else {
			label := "Parameters"
			fmt.Fprintf(&sb, "  %s (%d):\n", Dim.Render(label), len(m.items))
			maxShow := 10
			for i, item := range m.items {
				if i >= maxShow {
					remaining := len(m.items) - maxShow
					fmt.Fprintf(&sb, "    %s\n", Dim.Render(fmt.Sprintf("… and %d more", remaining)))
					break
				}
				fmt.Fprintf(&sb, "    %s %s\n", Dim.Render("•"), m.itemStyle().Render(item))
			}
		}
		sb.WriteString("\n")
	}

	// Icon based on style.
	icon := Cyan.Render("?")
	if m.style == ConfirmStyleDestructive {
		icon = Red.Render("⚠")
	}

	// Message.
	msgStyle := Bold
	if m.style == ConfirmStyleDestructive {
		msgStyle = Renderer.NewStyle().Bold(true).Foreground(Red.GetForeground())
	}

	// Toggle highlight for Y/N.
	var yLabel, nLabel string
	if m.focused {
		yLabel = Selected.Render("Y")
		nLabel = "n"
	} else {
		yLabel = Dim.Render("y")
		nLabel = Bold.Render("N")
	}
	hint := fmt.Sprintf("[%s/%s]", yLabel, nLabel)

	fmt.Fprintf(&sb, "%s %s %s", icon, msgStyle.Render(m.message), hint)

	// Help bar or overlay.
	if overlay := m.help.View(); overlay != "" {
		sb.WriteString("\n\n" + overlay)
	} else {
		sb.WriteString("  " + HelpBar(ConfirmKeys.Toggle, ConfirmKeys.Cancel))
	}

	return sb.String()
}

// itemStyle returns the appropriate style for listed items.
func (m ConfirmModel) itemStyle() lipgloss.Style {
	if m.style == ConfirmStyleDestructive {
		return Red
	}
	return Bold
}

// Confirmed returns whether the user confirmed the prompt.
func (m ConfirmModel) Confirmed() bool {
	return m.confirm
}

// Done returns whether the user has made a choice.
func (m ConfirmModel) Done() bool {
	return m.done
}

// RunConfirm runs a Y/N confirmation prompt and returns the result.
func RunConfirm(message string, opts ...ConfirmOption) (bool, error) {
	m := NewConfirm(message, opts...)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}
	return finalModel.(ConfirmModel).Confirmed(), nil
}

// RunDeleteConfirm runs a destructive confirmation prompt for a single item.
func RunDeleteConfirm(message string, itemName string) (bool, error) {
	return RunConfirm(message, WithDestructive(), WithItems([]string{itemName}))
}

// RunBatchDeleteConfirm runs a destructive confirmation prompt for multiple items.
func RunBatchDeleteConfirm(message string, items []string) (bool, error) {
	return RunConfirm(message, WithDestructive(), WithItems(items))
}
