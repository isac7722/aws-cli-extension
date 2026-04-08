package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSelector_DefaultCursor(t *testing.T) {
	items := []SelectorItem{
		{Label: "default", Value: "default"},
		{Label: "staging", Value: "staging"},
	}
	m := NewSelector(items, "Pick one:")
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
	if m.chosen != -1 {
		t.Errorf("expected chosen -1, got %d", m.chosen)
	}
}

func TestNewSelector_PreselectedCursor(t *testing.T) {
	items := []SelectorItem{
		{Label: "default", Value: "default"},
		{Label: "staging", Value: "staging", Selected: true},
		{Label: "prod", Value: "prod"},
	}
	m := NewSelector(items, "Pick one:")
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 (preselected), got %d", m.cursor)
	}
}

func TestSelectorModel_MoveDown(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
		{Label: "b", Value: "b"},
		{Label: "c", Value: "c"},
	}
	m := NewSelector(items, "")

	// Move down with 'j'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(SelectorModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after j, got %d", m.cursor)
	}

	// Move down with arrow key
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SelectorModel)
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2 after down, got %d", m.cursor)
	}

	// Cannot move past last item
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SelectorModel)
	if m.cursor != 2 {
		t.Errorf("expected cursor clamped at 2, got %d", m.cursor)
	}
}

func TestSelectorModel_MoveUp(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
		{Label: "b", Value: "b", Selected: true},
	}
	m := NewSelector(items, "")
	// cursor starts at 1 (preselected)

	// Move up with 'k'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(SelectorModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0 after k, got %d", m.cursor)
	}

	// Cannot move above first item
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SelectorModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor clamped at 0, got %d", m.cursor)
	}
}

func TestSelectorModel_EnterConfirms(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
		{Label: "b", Value: "b"},
	}
	m := NewSelector(items, "")

	// Move down then press enter
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SelectorModel)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SelectorModel)

	if m.Chosen() != 1 {
		t.Errorf("expected chosen=1, got %d", m.Chosen())
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd, got nil")
	}
}

func TestSelectorModel_EscCancels(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
	}
	m := NewSelector(items, "")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updated.(SelectorModel)

	if m.Chosen() != -1 {
		t.Errorf("expected chosen=-1 on cancel, got %d", m.Chosen())
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd, got nil")
	}
}

func TestSelectorModel_QCancels(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
	}
	m := NewSelector(items, "")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updated.(SelectorModel)

	if m.Chosen() != -1 {
		t.Errorf("expected chosen=-1 on q, got %d", m.Chosen())
	}
}

func TestSelectorModel_CtrlCCancels(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
	}
	m := NewSelector(items, "")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(SelectorModel)

	if m.Chosen() != -1 {
		t.Errorf("expected chosen=-1 on ctrl+c, got %d", m.Chosen())
	}
}

func TestSelectorModel_ViewContainsHeader(t *testing.T) {
	items := []SelectorItem{
		{Label: "default", Value: "default"},
	}
	m := NewSelector(items, "Select a profile:")
	view := m.View()

	if !strings.Contains(view, "Select a profile:") {
		t.Error("expected header in view output")
	}
}

func TestSelectorModel_ViewContainsLabels(t *testing.T) {
	items := []SelectorItem{
		{Label: "default", Value: "default"},
		{Label: "staging", Value: "staging", Hint: "us-west-2"},
	}
	m := NewSelector(items, "")
	view := m.View()

	if !strings.Contains(view, "default") {
		t.Error("expected 'default' label in view")
	}
	if !strings.Contains(view, "staging") {
		t.Error("expected 'staging' label in view")
	}
	if !strings.Contains(view, "us-west-2") {
		t.Error("expected hint 'us-west-2' in view")
	}
}

func TestSelectorModel_ViewShowsCursorIndicator(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
		{Label: "b", Value: "b"},
	}
	m := NewSelector(items, "")
	view := m.View()

	if !strings.Contains(view, "❯") {
		t.Error("expected cursor indicator ❯ in view")
	}
}

func TestSelectorModel_ViewShowsSelectedCheckmark(t *testing.T) {
	items := []SelectorItem{
		{Label: "default", Value: "default", Selected: true},
		{Label: "staging", Value: "staging"},
	}
	m := NewSelector(items, "")
	view := m.View()

	if !strings.Contains(view, "✔") {
		t.Error("expected checkmark ✔ for selected item in view")
	}
}

func TestSelectorModel_ViewShowsHelpText(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
	}
	m := NewSelector(items, "")
	view := m.View()

	if !strings.Contains(view, "select") {
		t.Error("expected help text with 'select' in view")
	}
	if !strings.Contains(view, "cancel") {
		t.Error("expected help text with 'cancel' in view")
	}
}

func TestSelectorModel_FormattedHint(t *testing.T) {
	items := []SelectorItem{
		{Label: "test", Value: "test", FormattedHint: "CUSTOM_HINT"},
	}
	m := NewSelector(items, "")
	view := m.View()

	if !strings.Contains(view, "CUSTOM_HINT") {
		t.Error("expected formatted hint in view")
	}
}

func TestSelectorModel_InitReturnsNil(t *testing.T) {
	items := []SelectorItem{{Label: "a", Value: "a"}}
	m := NewSelector(items, "")
	if cmd := m.Init(); cmd != nil {
		t.Error("expected Init to return nil cmd")
	}
}
