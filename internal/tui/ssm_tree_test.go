package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/isac7722/aws-cli-extension/internal/ssm"
)

// buildTestTree creates a test tree:
//
//	/ (root)
//	├── app/
//	│   ├── config/
//	│   │   ├── db_host (String)
//	│   │   └── db_pass (SecureString)
//	│   └── version (String)
//	└── shared/
//	    └── key (String)
func buildTestTree() *ssm.TreeNode {
	root := ssm.NewFolder("/", "/")

	app := ssm.NewFolder("app", "/app")
	config := ssm.NewFolder("config", "/app/config")
	config.AddChild(ssm.NewParameter("db_host", "/app/config/db_host", &ssm.ParameterMeta{Type: "String"}))
	config.AddChild(ssm.NewParameter("db_pass", "/app/config/db_pass", &ssm.ParameterMeta{Type: "SecureString"}))
	app.AddChild(config)
	app.AddChild(ssm.NewParameter("version", "/app/version", &ssm.ParameterMeta{Type: "String"}))

	shared := ssm.NewFolder("shared", "/shared")
	shared.AddChild(ssm.NewParameter("key", "/shared/key", &ssm.ParameterMeta{Type: "String"}))

	root.AddChild(app)
	root.AddChild(shared)

	return root
}

func TestNewSSMTree_InitialState(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "SSM Parameters")

	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
	if m.quit {
		t.Error("expected quit to be false")
	}
	if m.selected != nil {
		t.Error("expected selected to be nil")
	}
	// Initially only root children are visible (collapsed).
	if len(m.visible) != 2 {
		t.Errorf("expected 2 visible rows (app, shared), got %d", len(m.visible))
	}
}

func TestNewSSMTree_NilRoot(t *testing.T) {
	m := NewSSMTree(nil, "Empty")

	if len(m.visible) != 0 {
		t.Errorf("expected 0 visible rows for nil root, got %d", len(m.visible))
	}
}

func TestSSMTreeModel_Init(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init to return nil cmd")
	}
}

func TestSSMTreeModel_MoveDown(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Move down from app to shared.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(SSMTreeModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after j, got %d", m.cursor)
	}

	// Move down at bottom (should stay).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SSMTreeModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor clamped at 1, got %d", m.cursor)
	}
}

func TestSSMTreeModel_MoveUp(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")
	m.cursor = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(SSMTreeModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 after k, got %d", m.cursor)
	}

	// Move up at top (should stay).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SSMTreeModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor clamped at 0, got %d", m.cursor)
	}
}

func TestSSMTreeModel_ExpandCollapse_Enter(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Cursor on "app" folder — press enter to expand.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SSMTreeModel)

	if cmd != nil {
		t.Error("expanding a folder should not quit")
	}

	// After expanding "app": app, config, version, shared.
	if len(m.visible) != 4 {
		t.Errorf("expected 4 visible rows after expanding app, got %d", len(m.visible))
	}
	if m.visible[0].node.Name != "app" {
		t.Errorf("expected first row to be 'app', got %q", m.visible[0].node.Name)
	}
	if m.visible[1].node.Name != "config" {
		t.Errorf("expected second row to be 'config', got %q", m.visible[1].node.Name)
	}
	if m.visible[2].node.Name != "version" {
		t.Errorf("expected third row to be 'version', got %q", m.visible[2].node.Name)
	}

	// Press enter again to collapse.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SSMTreeModel)

	if len(m.visible) != 2 {
		t.Errorf("expected 2 visible rows after collapsing app, got %d", len(m.visible))
	}
}

func TestSSMTreeModel_ExpandCollapse_Space(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Space also toggles expand/collapse.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m = updated.(SSMTreeModel)

	// app expanded: app, config, version, shared.
	if len(m.visible) != 4 {
		t.Errorf("expected 4 visible rows after space expand, got %d", len(m.visible))
	}
}

func TestSSMTreeModel_ExpandWithRight(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Right/l should expand without toggle.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = updated.(SSMTreeModel)

	if len(m.visible) != 4 {
		t.Errorf("expected 4 visible rows after l expand, got %d", len(m.visible))
	}

	// Right again should NOT collapse.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = updated.(SSMTreeModel)

	if len(m.visible) != 4 {
		t.Errorf("expected still 4 visible rows after second l, got %d", len(m.visible))
	}
}

func TestSSMTreeModel_CollapseWithLeft(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Expand app first.
	m.visible[0].node.Expanded = true
	m.rebuildVisible()

	// Left/h on expanded folder should collapse.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m = updated.(SSMTreeModel)

	if len(m.visible) != 2 {
		t.Errorf("expected 2 visible rows after h collapse, got %d", len(m.visible))
	}
}

func TestSSMTreeModel_LeftJumpsToParent(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Expand app, then move cursor to "config" (depth 1, collapsed folder).
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 1 // "config" at depth 1

	// Left on collapsed child should jump to parent.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(SSMTreeModel)

	if m.cursor != 0 {
		t.Errorf("expected cursor to jump to parent (0), got %d", m.cursor)
	}
}

func TestSSMTreeModel_SelectParameter(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Expand app, then move to "version" (a parameter).
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 2 // "version" parameter

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SSMTreeModel)

	if cmd == nil {
		t.Error("selecting a parameter should quit")
	}
	if m.selected == nil {
		t.Fatal("expected selected to be set")
	}
	if m.selected.Name != "version" {
		t.Errorf("expected selected 'version', got %q", m.selected.Name)
	}
}

func TestSSMTreeModel_SelectedNode_NilOnCancel(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(SSMTreeModel)

	if m.SelectedNode() != nil {
		t.Error("expected SelectedNode() nil on cancel")
	}
}

func TestSSMTreeModel_EscQuits(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(SSMTreeModel)

	if !m.quit {
		t.Error("expected quit on esc")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd")
	}
}

func TestSSMTreeModel_QQuits(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updated.(SSMTreeModel)

	if !m.quit {
		t.Error("expected quit on q")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd")
	}
}

func TestSSMTreeModel_CtrlCQuits(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(SSMTreeModel)

	if !m.quit {
		t.Error("expected quit on ctrl+c")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd")
	}
}

func TestSSMTreeModel_CursorNode(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	node := m.CursorNode()
	if node == nil {
		t.Fatal("expected non-nil cursor node")
	}
	if node.Name != "app" {
		t.Errorf("expected cursor node 'app', got %q", node.Name)
	}
}

func TestSSMTreeModel_CursorNode_Empty(t *testing.T) {
	m := NewSSMTree(nil, "")

	node := m.CursorNode()
	if node != nil {
		t.Error("expected nil cursor node for empty tree")
	}
}

func TestSSMTreeModel_DeepExpansion(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Expand "app".
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	// Expand "config".
	m.visible[1].node.Expanded = true
	m.rebuildVisible()

	// app, config, db_host, db_pass, version, shared.
	if len(m.visible) != 6 {
		t.Errorf("expected 6 visible rows after deep expansion, got %d", len(m.visible))
	}

	// Verify depths.
	expectedDepths := []int{0, 1, 2, 2, 1, 0}
	for i, row := range m.visible {
		if row.depth != expectedDepths[i] {
			t.Errorf("row %d (%s): expected depth %d, got %d", i, row.node.Name, expectedDepths[i], row.depth)
		}
	}
}

func TestSSMTreeModel_CollapseClampsCursor(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Expand app, move cursor to "version" (index 2).
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 3 // "shared" at index 3

	// Collapse "app" from cursor position 0.
	m.cursor = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SSMTreeModel)

	// After collapse: app, shared — 2 items.
	if m.cursor >= len(m.visible) {
		t.Errorf("cursor %d should be within visible range %d", m.cursor, len(m.visible))
	}
}

// --- View rendering tests ---

func TestSSMTreeModel_ViewContainsHeader(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "SSM Parameter Store")

	view := m.View()
	if !strings.Contains(view, "SSM Parameter Store") {
		t.Error("expected view to contain header")
	}
}

func TestSSMTreeModel_ViewContainsNodeNames(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	view := m.View()
	if !strings.Contains(view, "app") {
		t.Error("expected view to contain 'app'")
	}
	if !strings.Contains(view, "shared") {
		t.Error("expected view to contain 'shared'")
	}
}

func TestSSMTreeModel_ViewShowsFolderIcons(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Collapsed folder should show ▸.
	view := m.View()
	if !strings.Contains(view, "▸") {
		t.Error("expected collapsed folder icon ▸ in view")
	}

	// Expand and check for ▾.
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	view = m.View()
	if !strings.Contains(view, "▾") {
		t.Error("expected expanded folder icon ▾ in view")
	}
}

func TestSSMTreeModel_ViewShowsCursor(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	view := m.View()
	if !strings.Contains(view, "❯") {
		t.Error("expected cursor indicator ❯ in view")
	}
}

func TestSSMTreeModel_ViewShowsTypeHints(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Expand to show parameters.
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	// Expand config.
	m.visible[1].node.Expanded = true
	m.rebuildVisible()

	view := m.View()
	if !strings.Contains(view, "[String]") {
		t.Error("expected type hint [String] in view")
	}
	if !strings.Contains(view, "[SecureString]") {
		t.Error("expected type hint [SecureString] in view")
	}
}

func TestSSMTreeModel_ViewShowsSecureLock(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Expand to show SecureString parameter.
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.visible[1].node.Expanded = true
	m.rebuildVisible()

	view := m.View()
	if !strings.Contains(view, "🔒") {
		t.Error("expected lock icon 🔒 for SecureString in view")
	}
}

func TestSSMTreeModel_ViewShowsParameterCount(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	view := m.View()
	// "app" has 3 parameters (db_host, db_pass, version).
	if !strings.Contains(view, "(3)") {
		t.Error("expected parameter count (3) for app folder")
	}
	// "shared" has 1 parameter (key).
	if !strings.Contains(view, "(1)") {
		t.Error("expected parameter count (1) for shared folder")
	}
}

func TestSSMTreeModel_ViewShowsHelpText(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	view := m.View()
	if !strings.Contains(view, "move") {
		t.Error("expected help text with 'move'")
	}
	if !strings.Contains(view, "cancel") {
		t.Error("expected help text with 'cancel'")
	}
	if !strings.Contains(view, "toggle") {
		t.Error("expected help text with 'toggle'")
	}
}

func TestSSMTreeModel_ViewEmptyTree(t *testing.T) {
	m := NewSSMTree(nil, "Empty")

	view := m.View()
	if !strings.Contains(view, "No parameters found") {
		t.Error("expected 'No parameters found' for nil root")
	}
}

func TestSSMTreeModel_ViewEmptyRootChildren(t *testing.T) {
	root := ssm.NewFolder("/", "/")
	m := NewSSMTree(root, "")

	view := m.View()
	if !strings.Contains(view, "No parameters found") {
		t.Error("expected 'No parameters found' for root with no children")
	}
}

func TestSSMTreeModel_ViewIndentation(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Expand app and config to get depth 2 nodes.
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.visible[1].node.Expanded = true
	m.rebuildVisible()

	view := m.View()
	lines := strings.Split(view, "\n")

	// Find lines with db_host (depth 2) — should have more indentation than app (depth 0).
	var dbHostLine, appLine string
	for _, line := range lines {
		if strings.Contains(line, "db_host") {
			dbHostLine = line
		}
		if strings.Contains(line, "app") && !strings.Contains(line, "db_") {
			appLine = line
		}
	}

	if dbHostLine == "" {
		t.Fatal("expected to find db_host line in view")
	}
	if appLine == "" {
		t.Fatal("expected to find app line in view")
	}

	// db_host line should be longer (more indentation) than app line.
	// This is a simple heuristic — deeper nodes have more leading spaces.
	dbHostTrimmed := strings.TrimLeft(dbHostLine, " ")
	appTrimmed := strings.TrimLeft(appLine, " ")
	dbHostIndent := len(dbHostLine) - len(dbHostTrimmed)
	appIndent := len(appLine) - len(appTrimmed)

	if dbHostIndent <= appIndent {
		t.Errorf("expected db_host to have more indentation (%d) than app (%d)", dbHostIndent, appIndent)
	}
}

// --- Detail panel tests ---

// mockFetchDetail returns a DetailFetchFunc that records calls and returns canned results.
func mockFetchDetail(detail *ssm.ParameterDetail, err error) (DetailFetchFunc, *[]string) {
	var calls []string
	fn := func(name string, decrypt bool) tea.Cmd {
		label := name
		if decrypt {
			label += ":decrypt"
		}
		calls = append(calls, label)
		return func() tea.Msg {
			return paramDetailMsg{detail: detail, err: err}
		}
	}
	return fn, &calls
}

func TestSSMTreeModel_VKeyOpensDetailOnParameter(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Expand app, move to "version" parameter.
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 2 // "version"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	if !m.ShowingDetail() {
		t.Error("expected detail panel to be open")
	}
	if m.detailNode == nil || m.detailNode.Name != "version" {
		t.Error("expected detail node to be 'version'")
	}
}

func TestSSMTreeModel_VKeyIgnoredOnFolder(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Cursor on "app" folder.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	if m.ShowingDetail() {
		t.Error("detail panel should not open on a folder")
	}
}

func TestSSMTreeModel_DetailPanelClosesOnV(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Open detail on a parameter.
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 2

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	if !m.ShowingDetail() {
		t.Fatal("expected detail panel to be open")
	}

	// Press v again to close.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	if m.ShowingDetail() {
		t.Error("expected detail panel to be closed after second v")
	}
}

func TestSSMTreeModel_DetailPanelClosesOnEsc(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 2

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	// Press esc to close detail (not quit the tree).
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(SSMTreeModel)

	if m.ShowingDetail() {
		t.Error("expected detail panel to be closed after esc")
	}
	if m.quit {
		t.Error("esc in detail panel should not quit the tree")
	}
	if cmd != nil {
		t.Error("esc in detail panel should not produce a quit cmd")
	}
}

func TestSSMTreeModel_DetailPanelQQuits(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 2

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	// q in detail panel should quit the entire program.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updated.(SSMTreeModel)

	if !m.quit {
		t.Error("expected quit on q in detail panel")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd")
	}
}

func TestSSMTreeModel_DetailWithFetcher(t *testing.T) {
	root := buildTestTree()
	detail := &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/version",
			Meta: &ssm.ParameterMeta{Type: "String", Version: 3, DataType: "text"},
		},
		Value: "1.2.3",
	}
	fetcher, calls := mockFetchDetail(detail, nil)
	m := NewSSMTreeWithFetcher(root, "", fetcher)

	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 2 // "version"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	if !m.ShowingDetail() {
		t.Fatal("expected detail panel open")
	}
	if !m.detailLoading {
		t.Error("expected detailLoading to be true")
	}
	if len(*calls) != 1 || (*calls)[0] != "/app/version" {
		t.Errorf("unexpected fetch calls: %v", *calls)
	}

	// Simulate the fetch completing.
	if cmd != nil {
		msg := cmd()
		updated, _ = m.Update(msg)
		m = updated.(SSMTreeModel)
	}

	if m.detailLoading {
		t.Error("expected detailLoading to be false after fetch")
	}
	if m.DetailInfo() == nil {
		t.Fatal("expected detail to be populated")
	}
	if m.DetailInfo().Value != "1.2.3" {
		t.Errorf("expected value '1.2.3', got %q", m.DetailInfo().Value)
	}
}

func TestSSMTreeModel_DetailFetchError(t *testing.T) {
	root := buildTestTree()
	fetcher, _ := mockFetchDetail(nil, fmt.Errorf("access denied"))
	m := NewSSMTreeWithFetcher(root, "", fetcher)

	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 2

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	// Simulate fetch error.
	if cmd != nil {
		msg := cmd()
		updated, _ = m.Update(msg)
		m = updated.(SSMTreeModel)
	}

	if m.DetailError() == nil {
		t.Fatal("expected detail error")
	}
	if !strings.Contains(m.DetailError().Error(), "access denied") {
		t.Errorf("unexpected error: %v", m.DetailError())
	}
}

func TestSSMTreeModel_DecryptSecureString(t *testing.T) {
	root := buildTestTree()

	// First fetch returns non-decrypted.
	callCount := 0
	fetcher := func(name string, decrypt bool) tea.Cmd {
		callCount++
		return func() tea.Msg {
			val := "••••••••"
			if decrypt {
				val = "s3cret"
			}
			return paramDetailMsg{
				detail: &ssm.ParameterDetail{
					FlatParam: ssm.FlatParam{
						Path: name,
						Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 1},
					},
					Value: val,
				},
			}
		}
	}

	m := NewSSMTreeWithFetcher(root, "", fetcher)

	// Expand app > config, move to "db_pass" (SecureString).
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.visible[1].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 3 // db_pass

	// Open detail.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	// Complete initial fetch.
	if cmd != nil {
		msg := cmd()
		updated, _ = m.Update(msg)
		m = updated.(SSMTreeModel)
	}

	if m.IsDecrypted() {
		t.Error("should not be decrypted yet")
	}

	// Press 'd' to decrypt.
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(SSMTreeModel)

	if !m.IsDecrypted() {
		t.Error("expected decrypted flag after pressing d")
	}

	// Complete decrypt fetch.
	if cmd != nil {
		msg := cmd()
		updated, _ = m.Update(msg)
		m = updated.(SSMTreeModel)
	}

	if m.DetailInfo().Value != "s3cret" {
		t.Errorf("expected decrypted value 's3cret', got %q", m.DetailInfo().Value)
	}
}

func TestSSMTreeModel_DecryptIgnoredOnNonSecureString(t *testing.T) {
	root := buildTestTree()
	callCount := 0
	fetcher := func(name string, decrypt bool) tea.Cmd {
		callCount++
		return func() tea.Msg {
			return paramDetailMsg{
				detail: &ssm.ParameterDetail{
					FlatParam: ssm.FlatParam{Path: name, Meta: &ssm.ParameterMeta{Type: "String"}},
					Value:     "hello",
				},
			}
		}
	}

	m := NewSSMTreeWithFetcher(root, "", fetcher)
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 2 // "version" (String type)

	// Open detail.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)
	if cmd != nil {
		msg := cmd()
		updated, _ = m.Update(msg)
		m = updated.(SSMTreeModel)
	}

	initialCalls := callCount

	// Press 'd' — should be ignored since it's not SecureString.
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(SSMTreeModel)

	if cmd != nil {
		t.Error("d should not trigger a fetch on non-SecureString")
	}
	if callCount != initialCalls {
		t.Error("no additional fetch should have been made")
	}
}

func TestSSMTreeModel_DetailViewRendersMetadata(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Add richer metadata to the version node.
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	versionNode := m.visible[2].node
	versionNode.Meta = &ssm.ParameterMeta{
		Type:     "String",
		Version:  5,
		DataType: "text",
		ARN:      "arn:aws:ssm:us-east-1:123456789:parameter/app/version",
	}

	m.cursor = 2
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	view := m.View()

	if !strings.Contains(view, "Parameter Detail") {
		t.Error("expected 'Parameter Detail' header in view")
	}
	if !strings.Contains(view, "version") {
		t.Error("expected parameter name in detail view")
	}
	if !strings.Contains(view, "/app/version") {
		t.Error("expected parameter path in detail view")
	}
	if !strings.Contains(view, "String") {
		t.Error("expected type in detail view")
	}
	if !strings.Contains(view, "5") {
		t.Error("expected version number in detail view")
	}
	if !strings.Contains(view, "text") {
		t.Error("expected data type in detail view")
	}
	if !strings.Contains(view, "arn:aws:ssm") {
		t.Error("expected ARN in detail view")
	}
}

func TestSSMTreeModel_DetailViewSecureStringMasked(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Navigate to SecureString parameter.
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.visible[1].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 3 // db_pass (SecureString)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	view := m.View()

	if !strings.Contains(view, "••••••••") {
		t.Error("expected masked value for SecureString")
	}
	if !strings.Contains(view, "decrypt") {
		t.Error("expected decrypt hint in view")
	}
}

func TestSSMTreeModel_DetailViewUpdatedHelp(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Verify normal help text.
	view := m.View()
	if !strings.Contains(view, "v: detail") {
		t.Error("expected 'v: detail' in normal help text")
	}

	// Open detail panel.
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 2

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	view = m.View()
	if !strings.Contains(view, "close detail") {
		t.Error("expected 'close detail' in detail help text")
	}
}

func TestSSMTreeModel_DetailViewSecureStringDecryptHelp(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	// Navigate to SecureString parameter.
	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.visible[1].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 3 // db_pass (SecureString)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	view := m.View()
	if !strings.Contains(view, "d: decrypt") {
		t.Error("expected 'd: decrypt value' hint for SecureString")
	}
}

func TestSSMTreeModel_NavigationBlockedInDetailPanel(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "")

	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 2 // "version"

	// Open detail.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	originalCursor := m.cursor

	// Try navigation keys — they should be ignored in detail mode.
	for _, key := range []string{"j", "k"} {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		m = updated.(SSMTreeModel)
		if m.cursor != originalCursor {
			t.Errorf("cursor should not move in detail panel (key=%s)", key)
		}
	}

	// Detail should still be open.
	if !m.ShowingDetail() {
		t.Error("detail panel should remain open after navigation key attempts")
	}
}

func TestSSMTreeModel_DetailWithoutFetcherUsesNodeMeta(t *testing.T) {
	root := buildTestTree()
	m := NewSSMTree(root, "") // No fetcher

	m.visible[0].node.Expanded = true
	m.rebuildVisible()
	m.cursor = 2 // "version"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(SSMTreeModel)

	if cmd != nil {
		t.Error("should not produce a cmd without a fetcher")
	}
	if !m.ShowingDetail() {
		t.Fatal("expected detail panel open")
	}
	if m.DetailInfo() == nil {
		t.Fatal("expected detail populated from node meta")
	}
	if m.DetailInfo().Path != "/app/version" {
		t.Errorf("expected path '/app/version', got %q", m.DetailInfo().Path)
	}
}

func TestNewSSMTreeWithFetcher(t *testing.T) {
	root := buildTestTree()
	fetcher := func(name string, decrypt bool) tea.Cmd { return nil }
	m := NewSSMTreeWithFetcher(root, "Test", fetcher)

	if m.fetchDetail == nil {
		t.Error("expected fetchDetail to be set")
	}
	if m.header != "Test" {
		t.Errorf("expected header 'Test', got %q", m.header)
	}
}
