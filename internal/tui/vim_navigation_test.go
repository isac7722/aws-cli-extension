package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/isac7722/aws-cli-extension/internal/ssm"
)

// TestVimNavigation_Selector_JK verifies vim j/k keys work in the selector.
func TestVimNavigation_Selector_JK(t *testing.T) {
	items := []SelectorItem{
		{Label: "first", Value: "1"},
		{Label: "second", Value: "2"},
		{Label: "third", Value: "3"},
	}
	m := NewSelector(items, "")

	// j moves cursor down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(SelectorModel)
	if m.cursor != 1 {
		t.Errorf("j: expected cursor=1, got %d", m.cursor)
	}

	// j again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(SelectorModel)
	if m.cursor != 2 {
		t.Errorf("j: expected cursor=2, got %d", m.cursor)
	}

	// k moves cursor up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(SelectorModel)
	if m.cursor != 1 {
		t.Errorf("k: expected cursor=1, got %d", m.cursor)
	}

	// k again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(SelectorModel)
	if m.cursor != 0 {
		t.Errorf("k: expected cursor=0, got %d", m.cursor)
	}
}

// TestVimNavigation_Selector_ArrowKeys verifies arrow keys work in the selector.
func TestVimNavigation_Selector_ArrowKeys(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
		{Label: "b", Value: "b"},
		{Label: "c", Value: "c"},
	}
	m := NewSelector(items, "")

	// Down arrow
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SelectorModel)
	if m.cursor != 1 {
		t.Errorf("down: expected cursor=1, got %d", m.cursor)
	}

	// Up arrow
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SelectorModel)
	if m.cursor != 0 {
		t.Errorf("up: expected cursor=0, got %d", m.cursor)
	}
}

// TestVimNavigation_Selector_BoundsClamping verifies j/k clamps at boundaries.
func TestVimNavigation_Selector_BoundsClamping(t *testing.T) {
	items := []SelectorItem{
		{Label: "only", Value: "only"},
	}
	m := NewSelector(items, "")

	// j on single item stays at 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(SelectorModel)
	if m.cursor != 0 {
		t.Errorf("j on single item: expected cursor=0, got %d", m.cursor)
	}

	// k on single item stays at 0
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(SelectorModel)
	if m.cursor != 0 {
		t.Errorf("k on single item: expected cursor=0, got %d", m.cursor)
	}
}

// TestVimNavigation_Selector_MixedJKAndArrows verifies interleaving j/k with arrow keys.
func TestVimNavigation_Selector_MixedJKAndArrows(t *testing.T) {
	items := []SelectorItem{
		{Label: "a", Value: "a"},
		{Label: "b", Value: "b"},
		{Label: "c", Value: "c"},
	}
	m := NewSelector(items, "")

	// j then down arrow then k then up arrow
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(SelectorModel)
	if m.cursor != 1 {
		t.Errorf("step 1 (j): expected 1, got %d", m.cursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SelectorModel)
	if m.cursor != 2 {
		t.Errorf("step 2 (down): expected 2, got %d", m.cursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(SelectorModel)
	if m.cursor != 1 {
		t.Errorf("step 3 (k): expected 1, got %d", m.cursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SelectorModel)
	if m.cursor != 0 {
		t.Errorf("step 4 (up): expected 0, got %d", m.cursor)
	}
}

// TestVimNavigation_SSMTree_JKHL verifies full vim navigation in SSM tree.
func TestVimNavigation_SSMTree_JKHL(t *testing.T) {
	root := &ssm.TreeNode{
		Name: "/",
		Path: "/",
		Children: []*ssm.TreeNode{
			{
				Name: "app",
				Path: "/app",
				Type: ssm.NodeFolder,
				Children: []*ssm.TreeNode{
					{Name: "key1", Path: "/app/key1"},
					{Name: "key2", Path: "/app/key2"},
				},
			},
			{Name: "standalone", Path: "/standalone"},
		},
	}

	m := NewSSMTree(root, "")
	// Initial visible: [app, standalone] (app is collapsed)
	if len(m.visible) != 2 {
		t.Fatalf("expected 2 visible rows, got %d", len(m.visible))
	}

	// j moves down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(SSMTreeModel)
	if m.cursor != 1 {
		t.Errorf("j: expected cursor=1, got %d", m.cursor)
	}

	// k moves back up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(SSMTreeModel)
	if m.cursor != 0 {
		t.Errorf("k: expected cursor=0, got %d", m.cursor)
	}

	// l expands folder
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = updated.(SSMTreeModel)
	if !root.Children[0].Expanded {
		t.Error("l: expected app folder to be expanded")
	}
	// After expansion: [app, key1, key2, standalone]
	if len(m.visible) != 4 {
		t.Errorf("after expand: expected 4 visible rows, got %d", len(m.visible))
	}

	// h collapses folder (cursor is on expanded folder)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m = updated.(SSMTreeModel)
	if root.Children[0].Expanded {
		t.Error("h: expected app folder to be collapsed")
	}
	if len(m.visible) != 2 {
		t.Errorf("after collapse: expected 2 visible rows, got %d", len(m.visible))
	}
}

// TestVimNavigation_SSMTree_ArrowKeys verifies arrow key navigation in SSM tree.
func TestVimNavigation_SSMTree_ArrowKeys(t *testing.T) {
	root := &ssm.TreeNode{
		Name: "/",
		Path: "/",
		Children: []*ssm.TreeNode{
			{
				Name: "folder",
				Path: "/folder",
				Type: ssm.NodeFolder,
				Children: []*ssm.TreeNode{
					{Name: "child", Path: "/folder/child"},
				},
			},
		},
	}

	m := NewSSMTree(root, "")

	// Right arrow expands
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(SSMTreeModel)
	if !root.Children[0].Expanded {
		t.Error("right: expected folder to be expanded")
	}

	// Down arrow moves into children
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SSMTreeModel)
	if m.cursor != 1 {
		t.Errorf("down: expected cursor=1, got %d", m.cursor)
	}

	// Left arrow on child jumps to parent
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(SSMTreeModel)
	if m.cursor != 0 {
		t.Errorf("left (jump to parent): expected cursor=0, got %d", m.cursor)
	}

	// Left arrow on expanded folder collapses it
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(SSMTreeModel)
	if root.Children[0].Expanded {
		t.Error("left: expected folder to be collapsed")
	}
}

// TestVimNavigation_SSMBrowser_JK verifies j/k in the SSM browser.
func TestVimNavigation_SSMBrowser_JK(t *testing.T) {
	m := NewSSMBrowser(SSMBrowserOptions{Prefix: "/"})
	// Simulate parameters loaded
	m.loading = false
	m.params = []SSMParam{
		{Name: "p1", Type: "String", Value: "v1"},
		{Name: "p2", Type: "String", Value: "v2"},
		{Name: "p3", Type: "String", Value: "v3"},
	}

	// j moves down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(SSMBrowserModel)
	if m.cursor != 1 {
		t.Errorf("j: expected cursor=1, got %d", m.cursor)
	}

	// down arrow also works
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SSMBrowserModel)
	if m.cursor != 2 {
		t.Errorf("down: expected cursor=2, got %d", m.cursor)
	}

	// k moves up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(SSMBrowserModel)
	if m.cursor != 1 {
		t.Errorf("k: expected cursor=1, got %d", m.cursor)
	}

	// up arrow also works
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SSMBrowserModel)
	if m.cursor != 0 {
		t.Errorf("up: expected cursor=0, got %d", m.cursor)
	}
}

// TestVimNavigation_KeyBindings_IncludeVimKeys verifies key binding definitions include vim keys.
func TestVimNavigation_KeyBindings_IncludeVimKeys(t *testing.T) {
	// Verify CommonKeys Up includes "k"
	upKeys := CommonKeys.Up.Help().Key
	if upKeys != "↑/k" {
		t.Errorf("CommonKeys.Up help key = %q, want contains 'k'", upKeys)
	}

	// Verify CommonKeys Down includes "j"
	downKeys := CommonKeys.Down.Help().Key
	if downKeys != "↓/j" {
		t.Errorf("CommonKeys.Down help key = %q, want contains 'j'", downKeys)
	}

	// Verify SelectorKeys Move includes jk
	moveKeys := SelectorKeys.Move.Help().Key
	if moveKeys != "↑↓/jk" {
		t.Errorf("SelectorKeys.Move help key = %q, want '↑↓/jk'", moveKeys)
	}

	// Verify SSMTreeKeys include h/l for expand/collapse
	expandKey := SSMTreeKeys.Expand.Help().Key
	if expandKey != "→/l" {
		t.Errorf("SSMTreeKeys.Expand help key = %q, want '→/l'", expandKey)
	}

	collapseKey := SSMTreeKeys.Collapse.Help().Key
	if collapseKey != "←/h" {
		t.Errorf("SSMTreeKeys.Collapse help key = %q, want '←/h'", collapseKey)
	}
}

// TestVimNavigation_ProfileEdit_DownUpNavigation verifies down/up navigate form fields.
func TestVimNavigation_ProfileEdit_DownUpNavigation(t *testing.T) {
	m := NewProfileEdit("Test", ProfileEditResult{})

	if m.focused != fieldProfileName {
		t.Fatalf("expected initial focus on fieldProfileName, got %d", m.focused)
	}

	// down moves to next field
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(ProfileEditModel)
	if m.focused != fieldAccessKeyID {
		t.Errorf("down: expected focus on fieldAccessKeyID(%d), got %d", fieldAccessKeyID, m.focused)
	}

	// up moves back
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(ProfileEditModel)
	if m.focused != fieldProfileName {
		t.Errorf("up: expected focus on fieldProfileName(%d), got %d", fieldProfileName, m.focused)
	}
}

// TestVimNavigation_SSMCreate_DownUpNavigation verifies down/up navigate form fields.
func TestVimNavigation_SSMCreate_DownUpNavigation(t *testing.T) {
	m := NewSSMCreate("Test", SSMCreateResult{})

	if m.focused != ssmFieldName {
		t.Fatalf("expected initial focus on ssmFieldName, got %d", m.focused)
	}

	// down moves to next field
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SSMCreateModel)
	if m.focused != ssmFieldValue {
		t.Errorf("down: expected focus on ssmFieldValue(%d), got %d", ssmFieldValue, m.focused)
	}

	// up moves back
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SSMCreateModel)
	if m.focused != ssmFieldName {
		t.Errorf("up: expected focus on ssmFieldName(%d), got %d", ssmFieldName, m.focused)
	}
}

// TestVimNavigation_SSMUpdate_DownUpNavigation verifies down/up navigate form fields.
func TestVimNavigation_SSMUpdate_DownUpNavigation(t *testing.T) {
	m := NewSSMUpdate("Test", SSMUpdateInput{
		Name:         "/test/param",
		CurrentValue: "old",
		Type:         "String",
		Version:      1,
	})

	if m.focused != ssmUpdateFieldValue {
		t.Fatalf("expected initial focus on ssmUpdateFieldValue, got %d", m.focused)
	}

	// down moves to next field
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SSMUpdateModel)
	if m.focused != ssmUpdateFieldType {
		t.Errorf("down: expected focus on ssmUpdateFieldType(%d), got %d", ssmUpdateFieldType, m.focused)
	}

	// up moves back
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SSMUpdateModel)
	if m.focused != ssmUpdateFieldValue {
		t.Errorf("up: expected focus on ssmUpdateFieldValue(%d), got %d", ssmUpdateFieldValue, m.focused)
	}
}

// TestVimNavigation_HelpTextShowsVimKeys verifies help bars mention vim keys.
func TestVimNavigation_HelpTextShowsVimKeys(t *testing.T) {
	items := []SelectorItem{{Label: "a", Value: "a"}}
	m := NewSelector(items, "")
	view := m.View()

	// The help bar should mention jk for movement
	if !containsAny(view, "jk", "j/k") {
		t.Error("selector help bar should mention vim j/k keys")
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
