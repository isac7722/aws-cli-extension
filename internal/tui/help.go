package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpEntry represents a single key-action pair for display in the help overlay.
type HelpEntry struct {
	Key         string
	Description string
}

// HelpEntryFromBinding creates a HelpEntry from a key.Binding.
func HelpEntryFromBinding(b key.Binding) HelpEntry {
	return HelpEntry{
		Key:         b.Help().Key,
		Description: b.Help().Desc,
	}
}

// HelpOverlayModel is a composable Bubble Tea component that renders a toggleable
// help overlay panel showing available key bindings. It is designed to be embedded
// in other TUI models and toggled with the '?' key.
type HelpOverlayModel struct {
	Visible bool
	title   string
	entries []HelpEntry
}

// NewHelpOverlay creates a new help overlay with the given title and entries.
func NewHelpOverlay(title string, entries []HelpEntry) HelpOverlayModel {
	return HelpOverlayModel{
		title:   title,
		entries: entries,
	}
}

// NewHelpOverlayFromBindings creates a help overlay from key.Binding values.
func NewHelpOverlayFromBindings(title string, bindings ...key.Binding) HelpOverlayModel {
	entries := make([]HelpEntry, 0, len(bindings))
	for _, b := range bindings {
		entries = append(entries, HelpEntryFromBinding(b))
	}
	return NewHelpOverlay(title, entries)
}

// Toggle switches the overlay visibility.
func (m *HelpOverlayModel) Toggle() {
	m.Visible = !m.Visible
}

// Update handles key events for the help overlay. Returns true if the event
// was consumed (i.e., the overlay was open and handled the key).
func (m *HelpOverlayModel) Update(msg tea.KeyMsg) (consumed bool) {
	switch msg.String() {
	case "?":
		m.Toggle()
		return true
	case "esc":
		if m.Visible {
			m.Visible = false
			return true
		}
	}
	return false
}

// View renders the help overlay panel. Returns empty string if not visible.
func (m HelpOverlayModel) View() string {
	if !m.Visible {
		return ""
	}
	return m.Render()
}

// Render renders the help overlay panel regardless of visibility.
// Useful for testing or when you want to force display.
func (m HelpOverlayModel) Render() string {
	var sb strings.Builder

	// Overlay border styles.
	borderStyle := Renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1)

	titleStyle := Renderer.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6"))

	keyStyle := Renderer.NewStyle().
		Foreground(lipgloss.Color("3")).
		Bold(true).
		Width(20)

	descStyle := Renderer.NewStyle().
		Foreground(lipgloss.Color("7"))

	// Title.
	sb.WriteString(titleStyle.Render(m.title) + "\n")
	sb.WriteString(Dim.Render(strings.Repeat("─", 30)) + "\n")

	// Entries.
	for _, entry := range m.entries {
		fmt.Fprintf(&sb, "%s %s\n",
			keyStyle.Render(entry.Key),
			descStyle.Render(entry.Description),
		)
	}

	sb.WriteString("\n" + Dim.Render("Press ? or esc to close"))

	return borderStyle.Render(sb.String())
}

// HelpBar renders a compact, single-line help bar from bindings.
// This is used at the bottom of TUI views as a quick-reference.
func HelpBar(bindings ...key.Binding) string {
	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		h := b.Help()
		if h.Key != "" && h.Desc != "" {
			parts = append(parts, h.Key+": "+h.Desc)
		}
	}
	return Dim.Render(strings.Join(parts, "  "))
}
