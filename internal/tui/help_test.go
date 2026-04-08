package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewHelpOverlay_DefaultHidden(t *testing.T) {
	h := NewHelpOverlay("Test Help", []HelpEntry{
		{Key: "enter", Description: "confirm"},
	})
	if h.Visible {
		t.Error("expected help overlay to be hidden by default")
	}
}

func TestHelpOverlay_Toggle(t *testing.T) {
	h := NewHelpOverlay("Test Help", []HelpEntry{
		{Key: "enter", Description: "confirm"},
	})

	h.Toggle()
	if !h.Visible {
		t.Error("expected help overlay to be visible after toggle")
	}

	h.Toggle()
	if h.Visible {
		t.Error("expected help overlay to be hidden after second toggle")
	}
}

func TestHelpOverlay_Update_QuestionMarkToggles(t *testing.T) {
	h := NewHelpOverlay("Test Help", []HelpEntry{
		{Key: "enter", Description: "confirm"},
	})

	consumed := h.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if !consumed {
		t.Error("expected '?' to be consumed by help overlay")
	}
	if !h.Visible {
		t.Error("expected help overlay to be visible after '?'")
	}

	consumed = h.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if !consumed {
		t.Error("expected second '?' to be consumed")
	}
	if h.Visible {
		t.Error("expected help overlay to be hidden after second '?'")
	}
}

func TestHelpOverlay_Update_EscClosesWhenVisible(t *testing.T) {
	h := NewHelpOverlay("Test Help", []HelpEntry{
		{Key: "enter", Description: "confirm"},
	})

	// Esc when hidden should not be consumed.
	consumed := h.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if consumed {
		t.Error("expected esc to not be consumed when overlay is hidden")
	}

	// Open the overlay.
	h.Toggle()

	// Esc when visible should close and be consumed.
	consumed = h.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !consumed {
		t.Error("expected esc to be consumed when overlay is visible")
	}
	if h.Visible {
		t.Error("expected help overlay to be hidden after esc")
	}
}

func TestHelpOverlay_Update_OtherKeysNotConsumed(t *testing.T) {
	h := NewHelpOverlay("Test Help", []HelpEntry{})

	consumed := h.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if consumed {
		t.Error("expected enter to not be consumed by help overlay")
	}

	consumed = h.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if consumed {
		t.Error("expected 'j' to not be consumed by help overlay")
	}
}

func TestHelpOverlay_View_EmptyWhenHidden(t *testing.T) {
	h := NewHelpOverlay("Test Help", []HelpEntry{
		{Key: "enter", Description: "confirm"},
	})

	if h.View() != "" {
		t.Error("expected View() to return empty string when hidden")
	}
}

func TestHelpOverlay_View_ContainsEntriesWhenVisible(t *testing.T) {
	h := NewHelpOverlay("Test Help", []HelpEntry{
		{Key: "⏎", Description: "confirm selection"},
		{Key: "esc", Description: "cancel"},
	})
	h.Visible = true

	view := h.View()
	if !strings.Contains(view, "Test Help") {
		t.Error("expected view to contain title 'Test Help'")
	}
	if !strings.Contains(view, "confirm selection") {
		t.Error("expected view to contain 'confirm selection'")
	}
	if !strings.Contains(view, "cancel") {
		t.Error("expected view to contain 'cancel'")
	}
	if !strings.Contains(view, "? or esc to close") {
		t.Error("expected view to contain close instructions")
	}
}

func TestHelpOverlay_Render_AlwaysRenders(t *testing.T) {
	h := NewHelpOverlay("Help", []HelpEntry{
		{Key: "q", Description: "quit"},
	})

	// Render works even when hidden.
	rendered := h.Render()
	if !strings.Contains(rendered, "Help") {
		t.Error("expected Render() to contain title even when hidden")
	}
	if !strings.Contains(rendered, "quit") {
		t.Error("expected Render() to contain 'quit' even when hidden")
	}
}

func TestNewHelpOverlayFromBindings(t *testing.T) {
	b1 := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("⏎", "select"),
	)
	b2 := key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	)

	h := NewHelpOverlayFromBindings("Bindings Test", b1, b2)
	h.Visible = true
	view := h.View()

	if !strings.Contains(view, "select") {
		t.Error("expected view to contain 'select' from binding")
	}
	if !strings.Contains(view, "quit") {
		t.Error("expected view to contain 'quit' from binding")
	}
}

func TestHelpEntryFromBinding(t *testing.T) {
	b := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("⏎", "confirm"),
	)
	entry := HelpEntryFromBinding(b)

	if entry.Key != "⏎" {
		t.Errorf("Key = %q, want %q", entry.Key, "⏎")
	}
	if entry.Description != "confirm" {
		t.Errorf("Description = %q, want %q", entry.Description, "confirm")
	}
}

func TestHelpBar(t *testing.T) {
	b1 := key.NewBinding(key.WithKeys("enter"), key.WithHelp("⏎", "select"))
	b2 := key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))

	bar := HelpBar(b1, b2)
	if !strings.Contains(bar, "select") {
		t.Error("expected help bar to contain 'select'")
	}
	if !strings.Contains(bar, "quit") {
		t.Error("expected help bar to contain 'quit'")
	}
}

func TestHelpBar_EmptyBinding(t *testing.T) {
	b := key.NewBinding(key.WithKeys("x"))
	bar := HelpBar(b)
	// Binding with no help text should not appear.
	if strings.Contains(bar, "x:") {
		t.Error("expected empty-help binding to be excluded from bar")
	}
}

// Integration tests: verify help overlay works within selector model.

func TestSelectorModel_HelpToggle(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
		{Label: "b", Value: "b"},
	}
	m := NewSelector(items, "")

	// Initially help is hidden.
	if m.help.Visible {
		t.Error("expected help to be hidden initially")
	}

	// Press '?' to open help.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(SelectorModel)
	if !m.help.Visible {
		t.Error("expected help to be visible after '?'")
	}

	// View should contain help overlay content.
	view := m.View()
	if !strings.Contains(view, "Selector Keys") {
		t.Error("expected view to contain 'Selector Keys' when help is open")
	}

	// Press '?' again to close.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(SelectorModel)
	if m.help.Visible {
		t.Error("expected help to be hidden after second '?'")
	}
}

func TestSelectorModel_HelpVisibleBlocksNavigation(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
		{Label: "b", Value: "b"},
	}
	m := NewSelector(items, "")

	// Open help.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(SelectorModel)

	// Esc should close help, not quit the selector.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(SelectorModel)
	if m.help.Visible {
		t.Error("expected help to close on esc")
	}
	if m.quit {
		t.Error("esc should close help overlay, not quit selector")
	}
	if cmd != nil {
		t.Error("expected no quit command when closing help")
	}
}

func TestSelectorModel_ViewContainsHelpHint(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
	}
	m := NewSelector(items, "")
	view := m.View()

	// The help bar should include the '?' hint.
	if !strings.Contains(view, "?") {
		t.Error("expected help bar to contain '?' key hint")
	}
}
