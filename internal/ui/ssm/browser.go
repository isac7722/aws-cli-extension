package ssm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/isac7722/aws-cli-extension/internal/ssm"
	"github.com/isac7722/aws-cli-extension/internal/tui"
)

// ---------- constants ----------

const (
	// defaultPrefix is the root path when no prefix is specified.
	defaultPrefix = "/"

	// maxVisibleRows is the maximum number of tree rows shown before scrolling.
	maxVisibleRows = 20

	// fetchTimeout is the maximum duration for an SSM API call.
	fetchTimeout = 30 * time.Second
)

// viewMode tracks which sub-view the browser is currently showing.
type viewMode int

const (
	// viewTree is the default tree-navigation view.
	viewTree viewMode = iota
	// viewDetail shows parameter detail (value, metadata).
	viewDetail
)

// ---------- message types ----------

// parametersLoadedMsg is sent when SSM parameters have been fetched from AWS.
type parametersLoadedMsg struct {
	params []ssm.FlatParam
	err    error
}

// parameterDetailMsg is sent when full parameter detail (metadata + value) is fetched.
type parameterDetailMsg struct {
	detail *ssm.ParameterDetail
	err    error
}

// parameterDeletedMsg is sent after a parameter is deleted.
type parameterDeletedMsg struct {
	path string
	err  error
}

// parameterSavedMsg is sent after a parameter is created or updated.
type parameterSavedMsg struct {
	path string
	err  error
}

// clipboardCopiedMsg is sent after a value is copied to clipboard.
type clipboardCopiedMsg struct {
	path string
	err  error
}

// ---------- BrowserOptions ----------

// BrowserOptions configures the SSM browser TUI.
type BrowserOptions struct {
	// Prefix is the starting SSM path prefix (default "/").
	Prefix string
	// Profile is the AWS profile name used for credential resolution.
	Profile string
	// Region is the AWS region for SSM API calls.
	Region string
	// AccessKeyID and SecretAccessKey allow direct credential injection.
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// ---------- visibleRow ----------

// visibleRow is a flattened representation of a tree node with its depth level,
// used for rendering and cursor navigation.
type visibleRow struct {
	node  *ssm.TreeNode
	depth int
}

// ---------- BrowserModel ----------

// BrowserModel is the Bubble Tea model for interactively browsing SSM Parameter
// Store parameters in a tree view. It loads parameters from AWS on init, builds
// a tree, and supports expand/collapse navigation, value inspection,
// SecureString reveal, and interactive filtering.
type BrowserModel struct {
	// options holds the configuration provided at creation time.
	options BrowserOptions

	// tree is the root of the parameter path hierarchy.
	tree *ssm.TreeNode

	// visible is the flattened list of currently visible rows (respecting
	// folder expand/collapse state and any active filter).
	visible []*visibleRow

	// cursor is the index into visible for the currently highlighted row.
	cursor int

	// offset is the scroll offset when the visible list exceeds maxVisibleRows.
	offset int

	// mode tracks which sub-view is active (tree vs. detail).
	mode viewMode

	// detail holds the parameter detail for the currently viewed parameter
	// (non-nil only in viewDetail mode).
	detail *ssm.ParameterDetail

	// showSecure indicates whether the SecureString value should be revealed
	// in the detail view. Reset to false whenever the detail target changes.
	showSecure bool

	// loading is true while the initial parameter list is being fetched.
	loading bool

	// loadingDetail is true while a single parameter detail is being fetched.
	loadingDetail bool

	// err holds any error encountered during parameter loading.
	err error

	// quit is set to true when the user explicitly cancels.
	quit bool

	// selected holds the tree node chosen by the user (enter on a parameter
	// in tree view). Nil if the user cancelled.
	selected *ssm.TreeNode

	// selectedValue holds the value of the selected parameter for output.
	selectedValue string

	// filtering is true when the user is actively typing a filter query.
	filtering bool

	// filterText is the current filter input string. When non-empty, the
	// visible list only shows nodes whose name or path contains this string
	// (case-insensitive).
	filterText string

	// statusMessage is a transient message shown at the bottom (e.g. "Copied to clipboard").
	statusMessage string

	// pendingAction tracks a multi-step action: "delete", "create-name", "create-value", "edit-value".
	pendingAction string

	// inputBuffer holds text being entered for create/edit operations.
	inputBuffer string

	// pendingPath holds the path for create/edit operations.
	pendingPath string
}

// NewBrowser creates a new SSM browser model with the given options.
// The model will connect to AWS and fetch parameters on Init.
func NewBrowser(opts BrowserOptions) BrowserModel {
	if opts.Prefix == "" {
		opts.Prefix = defaultPrefix
	}
	return BrowserModel{
		options: opts,
		loading: true,
		mode:    viewTree,
	}
}

// ---------- tea.Model interface ----------

// Init implements tea.Model. It creates the SSM client and kicks off the
// initial parameter fetch.
func (m BrowserModel) Init() tea.Cmd {
	return m.initClient
}

// Update implements tea.Model. It dispatches on message type and delegates
// to the appropriate sub-handler based on the current view mode.
func (m BrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case parametersLoadedMsg:
		return m.handleParametersLoaded(msg)

	case parameterDetailMsg:
		return m.handleParameterDetail(msg)

	case clipboardCopiedMsg:
		if msg.err != nil {
			m.statusMessage = tui.Red.Render("Clipboard error: " + msg.err.Error())
			return m, nil
		}
		m.statusMessage = tui.Green.Render("✔ Copied to clipboard: " + msg.path)
		return m, nil

	case parameterDeletedMsg:
		if msg.err != nil {
			m.statusMessage = tui.Red.Render("Delete error: " + msg.err.Error())
		} else {
			m.statusMessage = tui.Green.Render("✔ Deleted: " + msg.path)
			m.loading = true
			m.cursor = 0
			m.offset = 0
			m.visible = m.visible[:0]
			return m, m.initClient
		}
		return m, nil

	case parameterSavedMsg:
		if msg.err != nil {
			m.statusMessage = tui.Red.Render("Save error: " + msg.err.Error())
		} else {
			m.statusMessage = tui.Green.Render("✔ Saved: " + msg.path)
			m.loading = true
			m.cursor = 0
			m.offset = 0
			m.visible = m.visible[:0]
			return m, m.initClient
		}
		return m, nil

	case tea.KeyMsg:
		// If there's a fatal error, any key quits.
		if m.err != nil && m.mode == viewTree {
			m.quit = true
			return m, tea.Quit
		}

		// Pending action input mode intercepts all keys.
		if m.pendingAction != "" {
			return m.updatePendingAction(msg)
		}

		// Filter input mode intercepts all keys.
		if m.filtering {
			return m.updateFilter(msg)
		}

		switch m.mode {
		case viewTree:
			return m.updateTree(msg)
		case viewDetail:
			return m.updateDetail(msg)
		}
	}
	return m, nil
}

// View implements tea.Model. It renders the appropriate view based on mode.
func (m BrowserModel) View() string {
	switch m.mode {
	case viewDetail:
		return m.viewDetail()
	default:
		return m.viewTree()
	}
}

// ---------- message handlers ----------

func (m BrowserModel) handleParametersLoaded(msg parametersLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}

	m.tree = ssm.BuildTree(msg.params)
	m.cursor = 0
	m.offset = 0
	m.rebuildVisible()

	// Auto-expand root children if there's only one top-level folder.
	if m.tree != nil && len(m.tree.Children) == 1 && m.tree.Children[0].IsFolder() {
		m.tree.Children[0].Expanded = true
		m.rebuildVisible()
	}

	return m, nil
}

func (m BrowserModel) handleParameterDetail(msg parameterDetailMsg) (tea.Model, tea.Cmd) {
	m.loadingDetail = false
	if msg.err != nil {
		if m.pendingAction == "edit-value" {
			m.statusMessage = tui.Red.Render("Error: " + msg.err.Error())
			m.pendingAction = ""
			return m, nil
		}
		m.err = msg.err
		m.mode = viewTree
		return m, nil
	}

	// If we're in edit-value mode, pre-fill the input buffer with the current value.
	if m.pendingAction == "edit-value" {
		m.inputBuffer = msg.detail.Value
		m.statusMessage = ""
		return m, nil
	}

	m.detail = msg.detail
	m.mode = viewDetail
	m.showSecure = false
	return m, nil
}

// ---------- tree view update ----------

func (m BrowserModel) updateTree(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.loading {
		// While loading, only allow quit keys.
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
		return m, nil
	}

	// Clamp cursor to visible bounds after any refresh.
	if m.cursor >= len(m.visible) {
		if len(m.visible) > 0 {
			m.cursor = len(m.visible) - 1
		} else {
			m.cursor = 0
		}
	}

	// Clear transient status on any key press.
	m.statusMessage = ""

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.adjustScroll()
		}
	case "down", "j":
		if m.cursor < len(m.visible)-1 {
			m.cursor++
			m.adjustScroll()
		}
	case "enter", " ":
		if len(m.visible) > 0 {
			node := m.visible[m.cursor].node
			if node.IsFolder() {
				node.Expanded = !node.Expanded
				m.rebuildVisible()
				if m.cursor >= len(m.visible) {
					m.cursor = len(m.visible) - 1
				}
				m.adjustScroll()
			} else {
				// Parameter selected — fetch detail and switch to detail view.
				m.selected = node
				m.loadingDetail = true
				return m, m.fetchParameterDetail(node.Path, node.IsSecureString())
			}
		}
	case "l", "right":
		// Expand folder without toggling.
		if len(m.visible) > 0 {
			node := m.visible[m.cursor].node
			if node.IsFolder() && !node.Expanded {
				node.Expanded = true
				m.rebuildVisible()
			}
		}
	case "h", "left":
		// Collapse folder, or jump to parent folder.
		if len(m.visible) > 0 {
			row := m.visible[m.cursor]
			if row.node.IsFolder() && row.node.Expanded {
				row.node.Expanded = false
				m.rebuildVisible()
				if m.cursor >= len(m.visible) {
					m.cursor = len(m.visible) - 1
				}
			} else if row.depth > 0 {
				m.jumpToParent()
			}
			m.adjustScroll()
		}
	case "y":
		// Copy parameter value to clipboard and select for stdout output.
		if len(m.visible) > 0 {
			node := m.visible[m.cursor].node
			if node.IsParameter() {
				m.selected = node
				m.statusMessage = tui.Dim.Render("Fetching value...")
				return m, m.fetchAndCopyValue(node.Path)
			}
		}
	case "n":
		// Create new parameter — enter name input mode.
		m.pendingAction = "create-name"
		m.inputBuffer = m.currentFolderPath()
		m.statusMessage = ""
		return m, nil
	case "e":
		// Edit parameter value.
		if len(m.visible) > 0 {
			node := m.visible[m.cursor].node
			if node.IsParameter() {
				m.pendingAction = "edit-value"
				m.pendingPath = node.Path
				m.inputBuffer = ""
				m.statusMessage = tui.Dim.Render("Fetching current value...")
				return m, m.fetchValueForEdit(node.Path)
			}
		}
	case "d":
		// Delete parameter — ask for confirmation.
		if len(m.visible) > 0 {
			node := m.visible[m.cursor].node
			if node.IsParameter() {
				m.pendingAction = "delete-confirm"
				m.pendingPath = node.Path
				m.statusMessage = ""
				return m, nil
			}
		}
	case "/":
		// Enter filter mode — user can type to narrow the tree.
		m.filtering = true
		m.filterText = ""
		return m, nil
	case "esc":
		// If a filter is active, clear it first; otherwise quit.
		if m.filterText != "" {
			m.filterText = ""
			m.rebuildVisible()
			m.cursor = 0
			m.offset = 0
			return m, nil
		}
		m.quit = true
		return m, tea.Quit
	case "q", "ctrl+c":
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

// ---------- filter mode update ----------

// updateFilter handles key events while the user is typing a filter query.
// Characters are appended to filterText; backspace removes the last character;
// enter or esc exits filter mode (enter keeps the filter, esc clears it).
func (m BrowserModel) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Confirm filter and return to tree navigation.
		m.filtering = false
		return m, nil
	case tea.KeyEsc:
		// Cancel filter — clear text and rebuild unfiltered.
		m.filtering = false
		m.filterText = ""
		m.rebuildVisible()
		m.cursor = 0
		m.offset = 0
		return m, nil
	case tea.KeyBackspace:
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.rebuildVisible()
			m.cursor = 0
			m.offset = 0
		}
		return m, nil
	case tea.KeyCtrlC:
		m.quit = true
		return m, tea.Quit
	case tea.KeyRunes:
		m.filterText += string(msg.Runes)
		m.rebuildVisible()
		m.cursor = 0
		m.offset = 0
		return m, nil
	}
	return m, nil
}

// ---------- detail view update ----------

func (m BrowserModel) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace", "q":
		// Return to tree view.
		m.mode = viewTree
		m.detail = nil
		m.showSecure = false
		m.err = nil
		return m, nil
	case "v":
		// Toggle SecureString value visibility.
		if m.detail != nil && m.detail.Meta != nil && m.detail.Meta.Type == "SecureString" {
			m.showSecure = !m.showSecure
			if m.showSecure && m.detail.Value == "" {
				// Need to fetch decrypted value.
				m.loadingDetail = true
				return m, m.fetchParameterDetail(m.detail.Path, true)
			}
		}
	case "y":
		// Copy value to clipboard and select for stdout output.
		if m.detail != nil && m.detail.Value != "" {
			m.selectedValue = m.detail.Value
			_ = clipboard.WriteAll(m.detail.Value)
			m.statusMessage = tui.Green.Render("✔ Copied to clipboard")
			return m, tea.Quit
		}
	case "ctrl+c":
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

// ---------- tree view rendering ----------

// breadcrumb builds a breadcrumb path string from the currently selected node's
// path. For example, if the cursor is on "/app/config/db_host", the breadcrumb
// renders as: / ❯ app ❯ config ❯ db_host
func (m BrowserModel) breadcrumb() string {
	if len(m.visible) == 0 || m.cursor >= len(m.visible) {
		return tui.Cyan.Render("/")
	}

	node := m.visible[m.cursor].node
	path := node.Path
	if path == "" || path == "/" {
		return tui.Cyan.Render("/")
	}

	// Split path into segments (e.g. "/app/config/db_host" → ["app", "config", "db_host"]).
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 {
		return tui.Cyan.Render("/")
	}

	sep := tui.Dim.Render(" ❯ ")
	parts := make([]string, 0, len(segments)+1)
	parts = append(parts, tui.Cyan.Render("/"))
	for i, seg := range segments {
		if i == len(segments)-1 {
			// Last segment is the current item — highlight it.
			parts = append(parts, tui.Selected.Render(seg))
		} else {
			parts = append(parts, tui.Dim.Render(seg))
		}
	}
	return strings.Join(parts, sep)
}

// statusInfo returns a status string with item counts and position.
func (m BrowserModel) statusInfo() string {
	if len(m.visible) == 0 {
		return ""
	}

	total := 0
	if m.tree != nil {
		total = m.tree.ParameterCount()
	}

	pos := fmt.Sprintf("%d/%d", m.cursor+1, len(m.visible))
	info := tui.Dim.Render(fmt.Sprintf("[%s · %d params]", pos, total))

	return info
}

func (m BrowserModel) viewTree() string {
	var sb strings.Builder

	// Title bar: SSM Parameter Store with profile and region context.
	header := fmt.Sprintf("SSM Parameter Store — %s [%s / %s]", m.options.Prefix, m.options.Profile, m.options.Region)
	sb.WriteString(tui.Dim.Render(header) + "\n")

	if m.loading {
		sb.WriteString("\n" + tui.Dim.Render("Loading parameters...") + "\n")
		return sb.String()
	}

	if m.err != nil {
		sb.WriteString("\n" + tui.Red.Render("Error: "+m.err.Error()) + "\n\n")
		sb.WriteString(tui.Dim.Render("Press any key to exit"))
		return sb.String()
	}

	if len(m.visible) == 0 {
		sb.WriteString("\n" + tui.Yellow.Render("No parameters found under "+m.options.Prefix) + "\n\n")
		sb.WriteString(tui.Dim.Render("esc/q: back"))
		return sb.String()
	}

	// Breadcrumb path display: shows the hierarchy leading to the current item.
	sb.WriteString(m.breadcrumb() + "  " + m.statusInfo() + "\n\n")

	// Determine visible window for scrolling.
	end := m.offset + maxVisibleRows
	if end > len(m.visible) {
		end = len(m.visible)
	}
	windowRows := m.visible[m.offset:end]

	// Show scroll indicator if needed.
	if m.offset > 0 {
		sb.WriteString(tui.Dim.Render("  ↑ more") + "\n")
	}

	for idx, row := range windowRows {
		globalIdx := m.offset + idx
		cursor := "  "
		if globalIdx == m.cursor {
			cursor = tui.Cursor.Render("❯ ")
		}

		indent := strings.Repeat("  ", row.depth)

		icon := ""
		if row.node.IsFolder() {
			if row.node.Expanded {
				icon = "▾ "
			} else {
				icon = "▸ "
			}
		} else {
			icon = "  "
		}

		label := row.node.Name
		if globalIdx == m.cursor {
			label = tui.Selected.Render(label)
		}

		// Type hint for parameters, child count for folders.
		hint := ""
		if row.node.IsParameter() && row.node.Meta != nil {
			hint = " " + tui.Dim.Render(fmt.Sprintf("[%s]", row.node.Meta.Type))
		}
		if row.node.IsFolder() {
			count := row.node.ParameterCount()
			hint = " " + tui.Dim.Render(fmt.Sprintf("(%d)", count))
		}

		// SecureString lock indicator.
		if row.node.IsSecureString() {
			hint += " " + tui.Yellow.Render("🔒")
		}

		fmt.Fprintf(&sb, "%s%s%s%s%s\n", cursor, indent, icon, label, hint)
	}

	if end < len(m.visible) {
		sb.WriteString(tui.Dim.Render("  ↓ more") + "\n")
	}

	// Loading detail indicator.
	if m.loadingDetail {
		sb.WriteString("\n" + tui.Dim.Render("Fetching parameter..."))
	}

	// Filter input or active filter indicator.
	if m.filtering {
		sb.WriteString("\n" + tui.Cyan.Render("/") + m.filterText + tui.Dim.Render("▏") + "\n")
	} else if m.filterText != "" {
		sb.WriteString("\n" + tui.Dim.Render("filter: ") + tui.Cyan.Render(m.filterText) + "\n")
	}

	// Pending action UI.
	if m.pendingAction != "" {
		sb.WriteString("\n")
		switch m.pendingAction {
		case "delete-confirm":
			sb.WriteString(tui.Red.Render("Delete ") + tui.Bold.Render(m.pendingPath) + tui.Red.Render("? ") + tui.Dim.Render("[y/N]"))
		case "create-name":
			sb.WriteString(tui.Cyan.Render("New parameter name: ") + m.inputBuffer + tui.Dim.Render("▏"))
		case "create-value":
			sb.WriteString(tui.Cyan.Render("Value for ") + tui.Bold.Render(m.pendingPath) + tui.Cyan.Render(": ") + m.inputBuffer + tui.Dim.Render("▏"))
		case "edit-value":
			sb.WriteString(tui.Cyan.Render("Edit ") + tui.Bold.Render(m.pendingPath) + tui.Cyan.Render(": ") + m.inputBuffer + tui.Dim.Render("▏"))
		}
		sb.WriteString("\n" + tui.Dim.Render("enter: confirm  esc: cancel"))
		return sb.String()
	}

	// Status message.
	if m.statusMessage != "" {
		sb.WriteString("\n" + m.statusMessage)
	}

	// Help bar: keyboard shortcuts.
	sb.WriteString("\n" + tui.Dim.Render("↑↓/jk: move  ⏎: select  /: filter  y: copy  n: new  e: edit  d: delete  q: quit"))

	return sb.String()
}

// ---------- detail view rendering ----------

func (m BrowserModel) viewDetail() string {
	var sb strings.Builder

	sb.WriteString(tui.Dim.Render("Parameter Detail") + "\n")

	if m.loadingDetail {
		sb.WriteString("\n" + tui.Dim.Render("Loading...") + "\n")
		return sb.String()
	}

	if m.detail == nil {
		sb.WriteString("\n" + tui.Yellow.Render("No detail available") + "\n")
		sb.WriteString("\n" + tui.Dim.Render("esc/backspace: back"))
		return sb.String()
	}

	// Breadcrumb path for detail view.
	detailPath := m.detail.Path
	segments := strings.Split(strings.Trim(detailPath, "/"), "/")
	sep := tui.Dim.Render(" ❯ ")
	crumbParts := []string{tui.Cyan.Render("/")}
	for i, seg := range segments {
		if i == len(segments)-1 {
			crumbParts = append(crumbParts, tui.Selected.Render(seg))
		} else {
			crumbParts = append(crumbParts, tui.Dim.Render(seg))
		}
	}
	sb.WriteString(strings.Join(crumbParts, sep) + "\n\n")

	// Path
	sb.WriteString(tui.Bold.Render("Path: ") + m.detail.Path + "\n")

	// Type
	if m.detail.Meta != nil {
		sb.WriteString(tui.Bold.Render("Type: ") + m.detail.Meta.Type + "\n")

		// Version
		sb.WriteString(tui.Bold.Render("Version: ") + fmt.Sprintf("%d", m.detail.Meta.Version) + "\n")

		// Last Modified
		if !m.detail.Meta.LastModified.IsZero() {
			sb.WriteString(tui.Bold.Render("Modified: ") + m.detail.Meta.LastModified.Format(time.RFC3339) + "\n")
		}

		// ARN
		if m.detail.Meta.ARN != "" {
			sb.WriteString(tui.Bold.Render("ARN: ") + tui.Dim.Render(m.detail.Meta.ARN) + "\n")
		}
	}

	// Value
	sb.WriteString("\n")
	if m.detail.Meta != nil && m.detail.Meta.Type == "SecureString" {
		if m.showSecure && m.detail.Value != "" {
			sb.WriteString(tui.Bold.Render("Value: ") + tui.Yellow.Render(m.detail.Value) + "\n")
			sb.WriteString(tui.Dim.Render("  (SecureString — revealed)") + "\n")
		} else {
			sb.WriteString(tui.Bold.Render("Value: ") + tui.Dim.Render("••••••••") + "\n")
			sb.WriteString(tui.Dim.Render("  Press 'v' to reveal SecureString value") + "\n")
		}
	} else if m.detail.Value != "" {
		sb.WriteString(tui.Bold.Render("Value: ") + tui.Cyan.Render(m.detail.Value) + "\n")
	} else {
		sb.WriteString(tui.Bold.Render("Value: ") + tui.Dim.Render("(empty)") + "\n")
	}

	// Status message.
	if m.statusMessage != "" {
		sb.WriteString("\n" + m.statusMessage)
	}

	sb.WriteString("\n" + tui.Dim.Render("v: reveal/hide  y: copy to clipboard  esc: back  q: quit"))

	return sb.String()
}

// ---------- commands (tea.Cmd) ----------

// initClient creates the SSM client and fetches parameters. This is the Init command.
func (m BrowserModel) initClient() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	client, err := ssm.NewClient(ctx, ssm.ClientOptions{
		Profile:         m.options.Profile,
		Region:          m.options.Region,
		AccessKeyID:     m.options.AccessKeyID,
		SecretAccessKey: m.options.SecretAccessKey,
		SessionToken:    m.options.SessionToken,
	})
	if err != nil {
		return parametersLoadedMsg{err: fmt.Errorf("failed to create SSM client: %w", err)}
	}

	params, err := client.ListParameters(ctx, m.options.Prefix)
	if err != nil {
		return parametersLoadedMsg{err: fmt.Errorf("failed to list parameters: %w", err)}
	}

	return parametersLoadedMsg{params: params}
}

// fetchParameterDetail returns a tea.Cmd that fetches full detail for a parameter.
func (m BrowserModel) fetchParameterDetail(path string, decrypt bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := ssm.NewClient(ctx, ssm.ClientOptions{
			Profile:         m.options.Profile,
			Region:          m.options.Region,
			AccessKeyID:     m.options.AccessKeyID,
			SecretAccessKey: m.options.SecretAccessKey,
			SessionToken:    m.options.SessionToken,
		})
		if err != nil {
			return parameterDetailMsg{err: fmt.Errorf("failed to create SSM client: %w", err)}
		}

		detail, err := client.GetParameterDetail(ctx, path, decrypt)
		if err != nil {
			return parameterDetailMsg{err: err}
		}

		return parameterDetailMsg{detail: detail}
	}
}

// ---------- tree navigation helpers ----------

// rebuildVisible flattens the tree into the visible slice based on expansion
// state and the current filter text.
func (m *BrowserModel) rebuildVisible() {
	m.visible = m.visible[:0]
	if m.tree == nil {
		return
	}

	if m.filterText == "" {
		// No filter — normal tree walk.
		for _, child := range m.tree.Children {
			m.flattenNode(child, 0)
		}
	} else {
		// Filter active — show all nodes that match (or have matching descendants),
		// expanding the tree to reveal matches.
		for _, child := range m.tree.Children {
			m.flattenNodeFiltered(child, 0)
		}
	}
}

// flattenNode recursively appends visible nodes at the given depth (no filter).
func (m *BrowserModel) flattenNode(node *ssm.TreeNode, depth int) {
	m.visible = append(m.visible, &visibleRow{node: node, depth: depth})
	if node.IsFolder() && node.Expanded {
		for _, child := range node.Children {
			m.flattenNode(child, depth+1)
		}
	}
}

// flattenNodeFiltered recursively appends nodes that match the filter or have
// matching descendants. Folders are automatically shown (expanded) when they
// contain matching children.
func (m *BrowserModel) flattenNodeFiltered(node *ssm.TreeNode, depth int) {
	if node.IsParameter() {
		if m.matchesFilter(node) {
			m.visible = append(m.visible, &visibleRow{node: node, depth: depth})
		}
		return
	}

	// Folder: check if any descendant matches.
	if !m.subtreeMatchesFilter(node) {
		return
	}

	m.visible = append(m.visible, &visibleRow{node: node, depth: depth})
	for _, child := range node.Children {
		m.flattenNodeFiltered(child, depth+1)
	}
}

// matchesFilter returns true if the node's name or path contains the filter
// text (case-insensitive).
func (m *BrowserModel) matchesFilter(node *ssm.TreeNode) bool {
	if m.filterText == "" {
		return true
	}
	lower := strings.ToLower(m.filterText)
	return strings.Contains(strings.ToLower(node.Name), lower) ||
		strings.Contains(strings.ToLower(node.Path), lower)
}

// subtreeMatchesFilter returns true if the node itself or any descendant
// matches the current filter.
func (m *BrowserModel) subtreeMatchesFilter(node *ssm.TreeNode) bool {
	if m.matchesFilter(node) {
		return true
	}
	for _, child := range node.Children {
		if m.subtreeMatchesFilter(child) {
			return true
		}
	}
	return false
}

// adjustScroll ensures the cursor is within the visible scroll window.
func (m *BrowserModel) adjustScroll() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+maxVisibleRows {
		m.offset = m.cursor - maxVisibleRows + 1
	}
}

// jumpToParent moves the cursor to the nearest ancestor folder at a lower depth.
func (m *BrowserModel) jumpToParent() {
	if m.cursor <= 0 {
		return
	}
	currentDepth := m.visible[m.cursor].depth
	for i := m.cursor - 1; i >= 0; i-- {
		if m.visible[i].depth < currentDepth && m.visible[i].node.IsFolder() {
			m.cursor = i
			return
		}
	}
}

// ---------- pending action handler ----------

// updatePendingAction handles input for create/edit/delete confirmation.
func (m BrowserModel) updatePendingAction(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.pendingAction {
	case "delete-confirm":
		switch msg.String() {
		case "y", "Y":
			path := m.pendingPath
			m.pendingAction = ""
			m.pendingPath = ""
			m.statusMessage = tui.Dim.Render("Deleting...")
			return m, m.deleteParameter(path)
		default:
			m.pendingAction = ""
			m.pendingPath = ""
			m.statusMessage = tui.Dim.Render("Cancelled")
			return m, nil
		}

	case "create-name":
		switch msg.Type {
		case tea.KeyEnter:
			if m.inputBuffer != "" {
				m.pendingPath = m.inputBuffer
				m.pendingAction = "create-value"
				m.inputBuffer = ""
			}
			return m, nil
		case tea.KeyEsc:
			m.pendingAction = ""
			m.inputBuffer = ""
			return m, nil
		case tea.KeyBackspace:
			if len(m.inputBuffer) > 0 {
				m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
			}
			return m, nil
		case tea.KeyCtrlC:
			m.quit = true
			return m, tea.Quit
		case tea.KeyRunes:
			m.inputBuffer += string(msg.Runes)
			return m, nil
		}

	case "create-value":
		switch msg.Type {
		case tea.KeyEnter:
			if m.inputBuffer != "" {
				path := m.pendingPath
				value := m.inputBuffer
				m.pendingAction = ""
				m.pendingPath = ""
				m.inputBuffer = ""
				m.statusMessage = tui.Dim.Render("Creating...")
				return m, m.putParameter(path, value)
			}
			return m, nil
		case tea.KeyEsc:
			m.pendingAction = ""
			m.pendingPath = ""
			m.inputBuffer = ""
			return m, nil
		case tea.KeyBackspace:
			if len(m.inputBuffer) > 0 {
				m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
			}
			return m, nil
		case tea.KeyCtrlC:
			m.quit = true
			return m, tea.Quit
		case tea.KeyRunes:
			m.inputBuffer += string(msg.Runes)
			return m, nil
		}

	case "edit-value":
		switch msg.Type {
		case tea.KeyEnter:
			if m.inputBuffer != "" {
				path := m.pendingPath
				value := m.inputBuffer
				m.pendingAction = ""
				m.pendingPath = ""
				m.inputBuffer = ""
				m.statusMessage = tui.Dim.Render("Saving...")
				return m, m.putParameter(path, value)
			}
			return m, nil
		case tea.KeyEsc:
			m.pendingAction = ""
			m.pendingPath = ""
			m.inputBuffer = ""
			return m, nil
		case tea.KeyBackspace:
			if len(m.inputBuffer) > 0 {
				m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
			}
			return m, nil
		case tea.KeyCtrlC:
			m.quit = true
			return m, tea.Quit
		case tea.KeyRunes:
			m.inputBuffer += string(msg.Runes)
			return m, nil
		}
	}
	return m, nil
}

// ---------- CRUD commands ----------

// fetchAndCopyValue fetches a parameter value, copies to clipboard, and selects for output.
func (m BrowserModel) fetchAndCopyValue(path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := ssm.NewClient(ctx, ssm.ClientOptions{
			Profile:         m.options.Profile,
			Region:          m.options.Region,
			AccessKeyID:     m.options.AccessKeyID,
			SecretAccessKey: m.options.SecretAccessKey,
			SessionToken:    m.options.SessionToken,
		})
		if err != nil {
			return clipboardCopiedMsg{path: path, err: err}
		}

		detail, err := client.GetParameterDetail(ctx, path, true)
		if err != nil {
			return clipboardCopiedMsg{path: path, err: err}
		}

		if err := clipboard.WriteAll(detail.Value); err != nil {
			return clipboardCopiedMsg{path: path, err: err}
		}

		return clipboardCopiedMsg{path: path}
	}
}

// fetchValueForEdit fetches a parameter value and pre-fills the input buffer.
func (m BrowserModel) fetchValueForEdit(path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := ssm.NewClient(ctx, ssm.ClientOptions{
			Profile:         m.options.Profile,
			Region:          m.options.Region,
			AccessKeyID:     m.options.AccessKeyID,
			SecretAccessKey: m.options.SecretAccessKey,
			SessionToken:    m.options.SessionToken,
		})
		if err != nil {
			return parameterSavedMsg{path: path, err: err}
		}

		detail, err := client.GetParameterDetail(ctx, path, true)
		if err != nil {
			return parameterSavedMsg{path: path, err: err}
		}

		// Return the detail so we can pre-fill the input buffer.
		return parameterDetailMsg{detail: detail}
	}
}

func (m BrowserModel) deleteParameter(path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := ssm.NewClient(ctx, ssm.ClientOptions{
			Profile:         m.options.Profile,
			Region:          m.options.Region,
			AccessKeyID:     m.options.AccessKeyID,
			SecretAccessKey: m.options.SecretAccessKey,
			SessionToken:    m.options.SessionToken,
		})
		if err != nil {
			return parameterDeletedMsg{path: path, err: err}
		}

		if err := client.DeleteParameter(ctx, path); err != nil {
			return parameterDeletedMsg{path: path, err: err}
		}

		return parameterDeletedMsg{path: path}
	}
}

func (m BrowserModel) putParameter(path, value string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := ssm.NewClient(ctx, ssm.ClientOptions{
			Profile:         m.options.Profile,
			Region:          m.options.Region,
			AccessKeyID:     m.options.AccessKeyID,
			SecretAccessKey: m.options.SecretAccessKey,
			SessionToken:    m.options.SessionToken,
		})
		if err != nil {
			return parameterSavedMsg{path: path, err: err}
		}

		_, err = client.PutParameter(ctx, ssm.PutParameterInput{
			Name:      path,
			Value:     value,
			Type:      "String",
			Overwrite: true,
		})

		return parameterSavedMsg{path: path, err: err}
	}
}

// currentFolderPath returns the path prefix of the currently selected folder or parameter's parent.
func (m *BrowserModel) currentFolderPath() string {
	if len(m.visible) == 0 {
		return m.options.Prefix
	}
	node := m.visible[m.cursor].node
	if node.IsFolder() {
		return node.Path + "/"
	}
	// Parent folder path
	idx := strings.LastIndex(node.Path, "/")
	if idx >= 0 {
		return node.Path[:idx+1]
	}
	return "/"
}

// ---------- public accessors ----------

// SelectedNode returns the parameter node chosen by the user, or nil if cancelled.
func (m BrowserModel) SelectedNode() *ssm.TreeNode {
	if m.quit {
		return nil
	}
	return m.selected
}

// SelectedValue returns the value of the selected parameter for stdout output.
// Returns "" if the user cancelled or no value was selected.
func (m BrowserModel) SelectedValue() string {
	if m.quit {
		return ""
	}
	return m.selectedValue
}

// IsQuit returns true if the user explicitly cancelled the browser.
func (m BrowserModel) IsQuit() bool {
	return m.quit
}

// RunBrowser runs the SSM browser TUI and returns the selected parameter node.
// Returns nil if the user cancelled. Renders to stderr so stdout stays clean
// for shell wrappers (eval $(awse ...)).
func RunBrowser(opts BrowserOptions) (*ssm.TreeNode, string, error) {
	m := NewBrowser(opts)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		return nil, "", fmt.Errorf("SSM browser error: %w", err)
	}
	if finalModel == nil {
		return nil, "", nil
	}
	browser, ok := finalModel.(BrowserModel)
	if !ok {
		return nil, "", nil
	}
	return browser.SelectedNode(), browser.SelectedValue(), nil
}
