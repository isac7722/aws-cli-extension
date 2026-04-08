package ssm

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/isac7722/aws-cli-extension/internal/ssm"
)

// ---------- helpers ----------

// buildTestTree creates a tree with both String and SecureString parameters
// for testing the browser's masking behavior.
func buildTestTree() *ssm.TreeNode {
	return ssm.BuildTree([]ssm.FlatParam{
		{
			Path: "/app/config/db_host",
			Meta: &ssm.ParameterMeta{Type: "String", Version: 1},
		},
		{
			Path: "/app/config/db_password",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 2},
		},
		{
			Path: "/app/config/api_key",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 1},
		},
		{
			Path: "/app/config/region",
			Meta: &ssm.ParameterMeta{Type: "String", Version: 1},
		},
	})
}

// newTestBrowser creates a BrowserModel preloaded with test parameters,
// bypassing the AWS client initialization.
func newTestBrowser() BrowserModel {
	m := NewBrowser(BrowserOptions{
		Prefix:  "/",
		Profile: "test",
		Region:  "us-east-1",
	})
	m.loading = false
	m.tree = buildTestTree()
	// Expand root children to make parameters visible.
	for _, child := range m.tree.Children {
		child.Expanded = true
		for _, grandchild := range child.Children {
			grandchild.Expanded = true
		}
	}
	m.rebuildVisible()
	return m
}

// findVisibleIndex returns the index of the first visible row whose path contains substr.
func findVisibleIndex(m BrowserModel, substr string) int {
	for i, row := range m.visible {
		if strings.Contains(row.node.Path, substr) {
			return i
		}
	}
	return -1
}

// sendKey sends a key message to the model and returns the updated model.
func sendKey(m BrowserModel, key string) BrowserModel {
	var msg tea.KeyMsg
	switch key {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
	case "ctrl+c":
		msg = tea.KeyMsg{Type: tea.KeyCtrlC}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	updated, _ := m.Update(msg)
	return updated.(BrowserModel)
}

// ---------- tests: SecureString masking in tree view ----------

func TestBrowser_TreeView_SecureStringShowsLockIcon(t *testing.T) {
	m := newTestBrowser()
	view := m.viewTree()

	// SecureString parameters should show a lock icon in the tree view.
	if !strings.Contains(view, "🔒") {
		t.Error("expected tree view to show lock icon 🔒 for SecureString parameters")
	}
}

func TestBrowser_TreeView_SecureStringDoesNotShowValue(t *testing.T) {
	m := newTestBrowser()
	view := m.viewTree()

	// Tree view should never show actual SecureString values inline.
	if strings.Contains(view, "decrypted_password") {
		t.Error("tree view should not display SecureString values")
	}
}

// ---------- tests: SecureString masking in detail view ----------

func TestBrowser_DetailView_SecureStringMaskedByDefault(t *testing.T) {
	m := newTestBrowser()

	// Simulate entering detail view for a SecureString parameter.
	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_password",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 2},
		},
		Value: "super_secret_value",
	}
	m.mode = viewDetail
	m.showSecure = false

	view := m.viewDetail()

	// Value should be masked with dots.
	if !strings.Contains(view, "••••••••") {
		t.Error("expected SecureString value to be masked with '••••••••' by default")
	}

	// The actual value should NOT appear.
	if strings.Contains(view, "super_secret_value") {
		t.Error("SecureString value should not be visible when showSecure is false")
	}

	// Should show the reveal hint.
	if !strings.Contains(view, "reveal") || !strings.Contains(view, "v") {
		t.Error("expected hint to press 'v' to reveal SecureString value")
	}
}

func TestBrowser_DetailView_SecureStringRevealedWithV(t *testing.T) {
	m := newTestBrowser()

	// Set up detail view with a SecureString that has a decrypted value.
	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_password",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 2},
		},
		Value: "super_secret_value",
	}
	m.mode = viewDetail
	m.showSecure = false

	// Press 'v' to reveal.
	m = sendKey(m, "v")

	if !m.showSecure {
		t.Error("expected showSecure to be true after pressing 'v'")
	}

	view := m.viewDetail()

	// Now the value should be visible.
	if !strings.Contains(view, "super_secret_value") {
		t.Error("expected SecureString value to be visible after pressing 'v'")
	}

	// Should indicate it's revealed.
	if !strings.Contains(view, "revealed") {
		t.Error("expected 'revealed' indicator when SecureString is shown")
	}
}

func TestBrowser_DetailView_SecureStringToggleHidesAgain(t *testing.T) {
	m := newTestBrowser()

	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_password",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 2},
		},
		Value: "super_secret_value",
	}
	m.mode = viewDetail
	m.showSecure = true // Start revealed.

	// Press 'v' to hide again.
	m = sendKey(m, "v")

	if m.showSecure {
		t.Error("expected showSecure to be false after toggling 'v' again")
	}

	view := m.viewDetail()

	// Value should be masked again.
	if strings.Contains(view, "super_secret_value") {
		t.Error("SecureString value should be hidden after toggling 'v' again")
	}
	if !strings.Contains(view, "••••••••") {
		t.Error("expected masked display after toggling 'v' to hide")
	}
}

func TestBrowser_DetailView_StringValueAlwaysShown(t *testing.T) {
	m := newTestBrowser()

	// Set up detail view with a regular String parameter.
	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_host",
			Meta: &ssm.ParameterMeta{Type: "String", Version: 1},
		},
		Value: "localhost:5432",
	}
	m.mode = viewDetail
	m.showSecure = false

	view := m.viewDetail()

	// String values should always be visible (no masking).
	if !strings.Contains(view, "localhost:5432") {
		t.Error("expected String parameter value to be displayed immediately")
	}

	// Should NOT show the mask.
	if strings.Contains(view, "••••••••") {
		t.Error("String parameters should not be masked")
	}
}

func TestBrowser_DetailView_VKeyNoEffectOnNonSecureString(t *testing.T) {
	m := newTestBrowser()

	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_host",
			Meta: &ssm.ParameterMeta{Type: "String", Version: 1},
		},
		Value: "localhost:5432",
	}
	m.mode = viewDetail
	m.showSecure = false

	// Press 'v' on a non-SecureString — should have no effect.
	m = sendKey(m, "v")

	if m.showSecure {
		t.Error("pressing 'v' on a non-SecureString should not toggle showSecure")
	}
}

func TestBrowser_DetailView_ShowSecureResetsOnNewDetail(t *testing.T) {
	m := newTestBrowser()

	// Simulate having revealed a SecureString.
	m.showSecure = true
	m.mode = viewDetail

	// Simulate receiving a new parameter detail (navigating to a different parameter).
	detail := &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/api_key",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 1},
		},
		Value: "",
	}
	updated, _ := m.Update(parameterDetailMsg{detail: detail})
	m = updated.(BrowserModel)

	if m.showSecure {
		t.Error("showSecure should be reset to false when navigating to a new parameter")
	}
}

func TestBrowser_DetailView_SecureStringEmptyValueTriggersDecryptFetch(t *testing.T) {
	m := newTestBrowser()

	// SecureString with empty value (not yet decrypted).
	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/api_key",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 1},
		},
		Value: "", // Empty — needs decryption.
	}
	m.mode = viewDetail
	m.showSecure = false

	// Press 'v' — should toggle showSecure and trigger a fetch since Value is empty.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(BrowserModel)

	if !m.showSecure {
		t.Error("expected showSecure to be true after pressing 'v'")
	}
	if !m.loadingDetail {
		t.Error("expected loadingDetail to be true when value is empty and reveal requested")
	}
	if cmd == nil {
		t.Error("expected a command to fetch decrypted value")
	}
}

func TestBrowser_DetailView_SecureStringWithValueDoesNotRefetch(t *testing.T) {
	m := newTestBrowser()

	// SecureString with value already populated.
	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_password",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 2},
		},
		Value: "already_decrypted",
	}
	m.mode = viewDetail
	m.showSecure = false

	// Press 'v' — should toggle without fetching.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(BrowserModel)

	if !m.showSecure {
		t.Error("expected showSecure to be true after pressing 'v'")
	}
	if m.loadingDetail {
		t.Error("should not set loadingDetail when value is already available")
	}
	if cmd != nil {
		t.Error("should not return a fetch command when value is already populated")
	}
}

// ---------- tests: detail view escape back to tree ----------

func TestBrowser_DetailView_EscReturnsToTreeView(t *testing.T) {
	m := newTestBrowser()

	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_password",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 2},
		},
		Value: "secret",
	}
	m.mode = viewDetail
	m.showSecure = true

	m = sendKey(m, "esc")

	if m.mode != viewTree {
		t.Errorf("expected mode to be viewTree after esc, got %v", m.mode)
	}
	if m.showSecure {
		t.Error("showSecure should be reset when returning to tree view")
	}
	if m.detail != nil {
		t.Error("detail should be nil after returning to tree view")
	}
}

func TestBrowser_DetailView_BackspaceReturnsToTreeView(t *testing.T) {
	m := newTestBrowser()

	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_password",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 2},
		},
		Value: "secret",
	}
	m.mode = viewDetail
	m.showSecure = true

	m = sendKey(m, "backspace")

	if m.mode != viewTree {
		t.Errorf("expected mode to be viewTree after backspace, got %v", m.mode)
	}
	if m.showSecure {
		t.Error("showSecure should be reset when returning to tree view via backspace")
	}
}

// ---------- tests: tree view initial state ----------

func TestBrowser_NewBrowser_DefaultState(t *testing.T) {
	m := NewBrowser(BrowserOptions{
		Prefix:  "/",
		Profile: "test",
		Region:  "us-east-1",
	})

	if !m.loading {
		t.Error("expected loading to be true initially")
	}
	if m.showSecure {
		t.Error("expected showSecure to be false initially")
	}
	if m.mode != viewTree {
		t.Error("expected initial mode to be viewTree")
	}
	if m.quit {
		t.Error("expected quit to be false initially")
	}
}

func TestBrowser_NewBrowser_DefaultPrefix(t *testing.T) {
	m := NewBrowser(BrowserOptions{})

	if m.options.Prefix != "/" {
		t.Errorf("expected default prefix '/', got %q", m.options.Prefix)
	}
}

// ---------- tests: SecureString identification in tree ----------

func TestBrowser_TreeView_IdentifiesSecureStringParams(t *testing.T) {
	m := newTestBrowser()

	secureCount := 0
	for _, row := range m.visible {
		if row.node.IsSecureString() {
			secureCount++
		}
	}

	if secureCount != 2 {
		t.Errorf("expected 2 SecureString parameters in tree, got %d", secureCount)
	}
}

// ---------- tests: CLI get command SecureString masking ----------

func TestBrowser_GetParameterDetail_NonDecrypt(t *testing.T) {
	// Test that parameters fetched without decrypt show SecureString type.
	m := newTestBrowser()

	// Simulate receiving a non-decrypted SecureString detail.
	detail := &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_password",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 2},
		},
		Value: "", // Empty because not decrypted.
	}
	updated, _ := m.Update(parameterDetailMsg{detail: detail})
	m = updated.(BrowserModel)

	if m.mode != viewDetail {
		t.Error("expected mode to switch to viewDetail")
	}
	if m.showSecure {
		t.Error("showSecure should be false by default for new detail")
	}

	view := m.viewDetail()
	if !strings.Contains(view, "SecureString") {
		t.Error("expected detail view to show SecureString type")
	}
	if !strings.Contains(view, "••••••••") {
		t.Error("expected SecureString value to be masked")
	}
}

// ---------- tests: detail view help text ----------

func TestBrowser_DetailView_HelpShowsRevealHint(t *testing.T) {
	m := newTestBrowser()

	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_password",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 2},
		},
		Value: "secret",
	}
	m.mode = viewDetail

	view := m.viewDetail()

	if !strings.Contains(view, "reveal/hide") {
		t.Error("expected detail view help to show 'reveal/hide' hint")
	}
}

// ---------- tests: detail view value output (y key) ----------

func TestBrowser_DetailView_YKeyCopiesValue(t *testing.T) {
	m := newTestBrowser()

	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_host",
			Meta: &ssm.ParameterMeta{Type: "String", Version: 1},
		},
		Value: "localhost:5432",
	}
	m.mode = viewDetail

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updated.(BrowserModel)

	if m.selectedValue != "localhost:5432" {
		t.Errorf("expected selectedValue to be 'localhost:5432', got %q", m.selectedValue)
	}
	if cmd == nil {
		t.Error("expected quit command after y")
	}
}

func TestBrowser_DetailView_YKeyEmptyValueNoQuit(t *testing.T) {
	m := newTestBrowser()

	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_password",
			Meta: &ssm.ParameterMeta{Type: "SecureString", Version: 2},
		},
		Value: "", // Empty — can't copy.
	}
	m.mode = viewDetail

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updated.(BrowserModel)

	if m.selectedValue != "" {
		t.Error("expected no selected value when value is empty")
	}
	if cmd != nil {
		t.Error("should not quit when value is empty")
	}
}

// ---------- tests: parameters loaded ----------

func TestBrowser_HandleParametersLoaded_Success(t *testing.T) {
	m := NewBrowser(BrowserOptions{Prefix: "/", Profile: "test", Region: "us-east-1"})
	m.loading = true

	params := []ssm.FlatParam{
		{Path: "/app/db_host", Meta: &ssm.ParameterMeta{Type: "String"}},
		{Path: "/app/db_pass", Meta: &ssm.ParameterMeta{Type: "SecureString"}},
	}

	updated, _ := m.Update(parametersLoadedMsg{params: params})
	m = updated.(BrowserModel)

	if m.loading {
		t.Error("expected loading to be false after parameters loaded")
	}
	if m.tree == nil {
		t.Error("expected tree to be built")
	}
	if m.err != nil {
		t.Errorf("unexpected error: %v", m.err)
	}
}

func TestBrowser_HandleParametersLoaded_Error(t *testing.T) {
	m := NewBrowser(BrowserOptions{Prefix: "/", Profile: "test", Region: "us-east-1"})
	m.loading = true

	updated, _ := m.Update(parametersLoadedMsg{err: errTestBrowser})
	m = updated.(BrowserModel)

	if m.loading {
		t.Error("expected loading to be false")
	}
	if m.err == nil {
		t.Error("expected error to be set")
	}
}

var errTestBrowser = &testBrowserError{}

type testBrowserError struct{}

func (e *testBrowserError) Error() string { return "test browser error" }

// ---------- tests: tree view navigation ----------

func TestBrowser_TreeView_Navigation(t *testing.T) {
	m := newTestBrowser()

	if len(m.visible) == 0 {
		t.Fatal("expected visible rows in test browser")
	}

	initial := m.cursor

	// Move down
	m = sendKey(m, "j")
	if m.cursor != initial+1 {
		t.Errorf("expected cursor %d after j, got %d", initial+1, m.cursor)
	}

	// Move up
	m = sendKey(m, "k")
	if m.cursor != initial {
		t.Errorf("expected cursor %d after k, got %d", initial, m.cursor)
	}
}

// ---------- tests: quit behavior ----------

func TestBrowser_TreeView_QuitKey(t *testing.T) {
	m := newTestBrowser()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updated.(BrowserModel)

	if !m.quit {
		t.Error("expected quit to be true")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
	if m.SelectedNode() != nil {
		t.Error("expected nil selected node on quit")
	}
	if m.SelectedValue() != "" {
		t.Error("expected empty selected value on quit")
	}
}

func TestBrowser_IsQuit(t *testing.T) {
	m := newTestBrowser()
	if m.IsQuit() {
		t.Error("should not be quit initially")
	}

	m.quit = true
	if !m.IsQuit() {
		t.Error("should be quit after setting quit=true")
	}
}

// ---------- tests: view rendering ----------

func TestBrowser_View_TreeMode(t *testing.T) {
	m := newTestBrowser()
	view := m.View()

	if !strings.Contains(view, "SSM Parameter Store") {
		t.Error("expected view to contain SSM Parameter Store header")
	}
}

func TestBrowser_View_DetailMode(t *testing.T) {
	m := newTestBrowser()
	m.mode = viewDetail
	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_host",
			Meta: &ssm.ParameterMeta{Type: "String", Version: 1},
		},
		Value: "test-value",
	}

	view := m.View()

	if !strings.Contains(view, "Parameter Detail") {
		t.Error("expected view to contain Parameter Detail header")
	}
	if !strings.Contains(view, "test-value") {
		t.Error("expected view to contain parameter value")
	}
}

func TestBrowser_View_Loading(t *testing.T) {
	m := NewBrowser(BrowserOptions{Prefix: "/", Profile: "test", Region: "us-east-1"})
	view := m.View()

	if !strings.Contains(view, "Loading") {
		t.Error("expected view to show loading state")
	}
}

func TestBrowser_View_Error(t *testing.T) {
	m := newTestBrowser()
	m.err = errTestBrowser

	view := m.View()

	if !strings.Contains(view, "Error") {
		t.Error("expected view to show error")
	}
}

func TestBrowser_View_Empty(t *testing.T) {
	m := NewBrowser(BrowserOptions{Prefix: "/test", Profile: "test", Region: "us-east-1"})
	m.loading = false
	m.tree = ssm.BuildTree(nil)
	m.rebuildVisible()

	view := m.View()

	if !strings.Contains(view, "No parameters found") {
		t.Error("expected view to show no parameters message")
	}
}

// ---------- tests: detail view nil checks ----------

func TestBrowser_DetailView_NilDetail(t *testing.T) {
	m := newTestBrowser()
	m.mode = viewDetail
	m.detail = nil

	view := m.viewDetail()
	if !strings.Contains(view, "No detail available") {
		t.Error("expected 'No detail available' for nil detail")
	}
}

func TestBrowser_DetailView_LoadingDetail(t *testing.T) {
	m := newTestBrowser()
	m.mode = viewDetail
	m.loadingDetail = true

	view := m.viewDetail()
	if !strings.Contains(view, "Loading") {
		t.Error("expected loading indicator in detail view")
	}
}

// ---------- tests: detail view metadata rendering ----------

func TestBrowser_DetailView_ShowsMetadata(t *testing.T) {
	m := newTestBrowser()
	m.mode = viewDetail
	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/db_host",
			Meta: &ssm.ParameterMeta{
				Type:    "String",
				Version: 3,
				ARN:     "arn:aws:ssm:us-east-1:123456789:parameter/app/config/db_host",
			},
		},
		Value: "myhost.example.com",
	}

	view := m.viewDetail()

	if !strings.Contains(view, "Path:") {
		t.Error("expected detail view to show Path label")
	}
	if !strings.Contains(view, "/app/config/db_host") {
		t.Error("expected detail view to show parameter path")
	}
	if !strings.Contains(view, "Type:") {
		t.Error("expected detail view to show Type label")
	}
	if !strings.Contains(view, "Version:") {
		t.Error("expected detail view to show Version label")
	}
	if !strings.Contains(view, "ARN:") {
		t.Error("expected detail view to show ARN")
	}
}

// ---------- tests: SecureString in CLI get output ----------

func TestBrowser_DetailView_EmptyValue(t *testing.T) {
	m := newTestBrowser()
	m.mode = viewDetail
	m.detail = &ssm.ParameterDetail{
		FlatParam: ssm.FlatParam{
			Path: "/app/config/empty_param",
			Meta: &ssm.ParameterMeta{Type: "String", Version: 1},
		},
		Value: "",
	}

	view := m.viewDetail()
	if !strings.Contains(view, "(empty)") {
		t.Error("expected '(empty)' for parameter with empty value")
	}
}
