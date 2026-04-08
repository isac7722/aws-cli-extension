package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirmModel_DefaultView(t *testing.T) {
	m := NewConfirm("Delete this?")
	view := m.View()

	if !strings.Contains(view, "Delete this?") {
		t.Errorf("view should contain message, got: %s", view)
	}
	if !strings.Contains(view, "N") {
		t.Error("view should show N (default deny)")
	}
}

func TestConfirmModel_DestructiveView(t *testing.T) {
	m := NewConfirm("Delete this?", WithDestructive())
	view := m.View()

	if !strings.Contains(view, "Delete this?") {
		t.Errorf("view should contain message, got: %s", view)
	}
	// Destructive style uses ⚠ icon.
	if !strings.Contains(view, "⚠") {
		t.Error("destructive view should contain ⚠ icon")
	}
}

func TestConfirmModel_WithSingleItem(t *testing.T) {
	m := NewConfirm("Delete?", WithItems([]string{"/app/config/key"}))
	view := m.View()

	if !strings.Contains(view, "/app/config/key") {
		t.Errorf("view should contain item name, got: %s", view)
	}
	if !strings.Contains(view, "Parameter:") {
		t.Error("single item should show 'Parameter:' label")
	}
}

func TestConfirmModel_WithMultipleItems(t *testing.T) {
	items := []string{"/a", "/b", "/c"}
	m := NewConfirm("Delete?", WithItems(items))
	view := m.View()

	for _, item := range items {
		if !strings.Contains(view, item) {
			t.Errorf("view should contain item %q, got: %s", item, view)
		}
	}
	if !strings.Contains(view, "(3)") {
		t.Error("batch view should show item count")
	}
}

func TestConfirmModel_WithManyItemsTruncates(t *testing.T) {
	items := make([]string, 15)
	for i := range items {
		items[i] = "/param/" + string(rune('a'+i))
	}
	m := NewConfirm("Delete?", WithItems(items))
	view := m.View()

	if !strings.Contains(view, "and 5 more") {
		t.Errorf("view should truncate and show remaining count, got: %s", view)
	}
}

func TestConfirmModel_YKey(t *testing.T) {
	m := NewConfirm("Delete?")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model := updated.(ConfirmModel)

	if !model.Confirmed() {
		t.Error("pressing 'y' should confirm")
	}
	if !model.Done() {
		t.Error("pressing 'y' should mark done")
	}
	if cmd == nil {
		t.Error("pressing 'y' should return quit cmd")
	}
}

func TestConfirmModel_NKey(t *testing.T) {
	m := NewConfirm("Delete?")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model := updated.(ConfirmModel)

	if model.Confirmed() {
		t.Error("pressing 'n' should deny")
	}
	if !model.Done() {
		t.Error("pressing 'n' should mark done")
	}
	if cmd == nil {
		t.Error("pressing 'n' should return quit cmd")
	}
}

func TestConfirmModel_EscKey(t *testing.T) {
	m := NewConfirm("Delete?")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(ConfirmModel)

	if model.Confirmed() {
		t.Error("pressing esc should deny")
	}
	if !model.Done() {
		t.Error("pressing esc should mark done")
	}
}

func TestConfirmModel_CtrlC(t *testing.T) {
	m := NewConfirm("Delete?")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model := updated.(ConfirmModel)

	if model.Confirmed() {
		t.Error("pressing ctrl+c should deny")
	}
	if !model.Done() {
		t.Error("pressing ctrl+c should mark done")
	}
}

func TestConfirmModel_ToggleFocus(t *testing.T) {
	m := NewConfirm("Delete?")

	// Default: focused=false (N is highlighted).
	if m.focused {
		t.Error("default focus should be false (N)")
	}

	// Press right to toggle to Y.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	model := updated.(ConfirmModel)
	if !model.focused {
		t.Error("pressing right should toggle focus to Y")
	}

	// Press left to toggle back to N.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(ConfirmModel)
	if model.focused {
		t.Error("pressing left should toggle focus back to N")
	}
}

func TestConfirmModel_TabToggle(t *testing.T) {
	m := NewConfirm("Delete?")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := updated.(ConfirmModel)
	if !model.focused {
		t.Error("pressing tab should toggle focus")
	}
}

func TestConfirmModel_EnterWithFocusY(t *testing.T) {
	m := NewConfirm("Delete?")
	// Toggle to Y.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	// Press enter.
	updated, _ = updated.(ConfirmModel).Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(ConfirmModel)

	if !model.Confirmed() {
		t.Error("enter with Y focused should confirm")
	}
}

func TestConfirmModel_EnterWithFocusN(t *testing.T) {
	m := NewConfirm("Delete?")
	// Default focus is N, press enter.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(ConfirmModel)

	if model.Confirmed() {
		t.Error("enter with N focused should deny")
	}
}

func TestConfirmModel_HelpToggle(t *testing.T) {
	m := NewConfirm("Delete?")

	// Press ? to show help.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	model := updated.(ConfirmModel)

	if !model.help.Visible {
		t.Error("pressing ? should show help overlay")
	}

	view := model.View()
	if !strings.Contains(view, "Confirm Keys") {
		t.Errorf("help overlay should show title, got: %s", view)
	}
}

func TestConfirmModel_Init(t *testing.T) {
	m := NewConfirm("test")
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil")
	}
}

func TestConfirmModel_DestructiveWithItems(t *testing.T) {
	m := NewConfirm("Delete?", WithDestructive(), WithItems([]string{"/a", "/b"}))
	view := m.View()

	if !strings.Contains(view, "⚠") {
		t.Error("destructive style should show ⚠")
	}
	if !strings.Contains(view, "/a") || !strings.Contains(view, "/b") {
		t.Error("should show all items")
	}
}
