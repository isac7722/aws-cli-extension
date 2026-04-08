package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/isac7722/aws-cli-extension/internal/ssm"
)

// DetailFetchFunc is a function that fetches the full detail of an SSM parameter.
// It is called asynchronously as a tea.Cmd. The decrypt parameter controls whether
// SecureString values are decrypted.
type DetailFetchFunc func(name string, decrypt bool) tea.Cmd

// paramDetailMsg is sent when parameter detail has been fetched.
type paramDetailMsg struct {
	detail *ssm.ParameterDetail
	err    error
}

// SSMTreeModel is a Bubble Tea model for browsing SSM parameters as a tree.
// It supports cursor movement, expand/collapse of folders, and indented rendering.
// When the user presses 'v' on a parameter, a detail panel is shown with full
// metadata and value.
type SSMTreeModel struct {
	root     *ssm.TreeNode
	visible  []*visibleRow // flattened list of currently visible nodes
	cursor   int
	quit     bool
	selected *ssm.TreeNode // node chosen by the user (enter on a parameter)
	header   string

	// Detail panel state.
	showDetail    bool                 // whether the detail panel is visible
	detailNode    *ssm.TreeNode        // the node being displayed in the detail panel
	detail        *ssm.ParameterDetail // fetched detail (may be nil if not yet fetched)
	detailErr     error                // error from detail fetch
	detailLoading bool                 // whether a detail fetch is in progress
	decrypted     bool                 // whether the current detail value has been decrypted
	fetchDetail   DetailFetchFunc      // optional function to fetch parameter details

	// Help overlay.
	help HelpOverlayModel
}

// visibleRow is a flattened representation of a tree node with its depth level.
type visibleRow struct {
	node  *ssm.TreeNode
	depth int
}

// NewSSMTree creates a new SSM tree browser model.
// The root node is the top of the parameter hierarchy (typically "/").
func NewSSMTree(root *ssm.TreeNode, header string) SSMTreeModel {
	m := SSMTreeModel{
		root:   root,
		header: header,
		help: NewHelpOverlayFromBindings("SSM Tree Keys",
			SSMTreeKeys.Move,
			SSMTreeKeys.Expand,
			SSMTreeKeys.Collapse,
			SSMTreeKeys.Toggle,
			SSMTreeKeys.Detail,
			SSMTreeKeys.Cancel,
			SSMTreeKeys.Help,
		),
	}
	m.rebuildVisible()
	return m
}

// NewSSMTreeWithFetcher creates a new SSM tree browser model with a detail fetch function.
// The fetcher is called when the user requests parameter details (v key) and enables
// fetching the full value and metadata from AWS.
func NewSSMTreeWithFetcher(root *ssm.TreeNode, header string, fetcher DetailFetchFunc) SSMTreeModel {
	m := NewSSMTree(root, header)
	m.fetchDetail = fetcher
	return m
}

// Init implements tea.Model. No initial command is needed since the tree data
// is provided at construction time.
func (m SSMTreeModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It handles keyboard navigation,
// expand/collapse toggling, and detail panel interactions.
func (m SSMTreeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case paramDetailMsg:
		m.detailLoading = false
		if msg.err != nil {
			m.detailErr = msg.err
			return m, nil
		}
		m.detail = msg.detail
		return m, nil

	case tea.KeyMsg:
		// Let help overlay handle '?' and esc-while-open first.
		if m.help.Update(msg) {
			return m, nil
		}

		// If detail panel is open, handle detail-specific keys first.
		if m.showDetail {
			return m.updateDetailPanel(msg)
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.visible)-1 {
				m.cursor++
			}
		case "enter", " ":
			if len(m.visible) > 0 {
				node := m.visible[m.cursor].node
				if node.IsFolder() {
					node.Expanded = !node.Expanded
					m.rebuildVisible()
					// Clamp cursor if the list shrunk.
					if m.cursor >= len(m.visible) {
						m.cursor = len(m.visible) - 1
					}
				} else {
					// Parameter selected — exit with this node.
					m.selected = node
					return m, tea.Quit
				}
			}
		case "v":
			// Open detail panel for the current parameter node.
			if len(m.visible) > 0 {
				node := m.visible[m.cursor].node
				if node.IsParameter() {
					return m.openDetailPanel(node)
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
				} else if row.depth > 0 {
					// Jump cursor to parent folder.
					m.jumpToParent()
				}
			}
		case "esc", "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// openDetailPanel opens the detail panel for the given parameter node.
func (m SSMTreeModel) openDetailPanel(node *ssm.TreeNode) (tea.Model, tea.Cmd) {
	m.showDetail = true
	m.detailNode = node
	m.detail = nil
	m.detailErr = nil
	m.decrypted = false
	m.detailLoading = false

	// If we have a fetcher, start an async fetch (non-decrypted first).
	if m.fetchDetail != nil {
		m.detailLoading = true
		return m, m.fetchDetail(node.Path, false)
	}

	// Without a fetcher, build detail from the node's existing metadata.
	if node.Meta != nil {
		m.detail = &ssm.ParameterDetail{
			FlatParam: ssm.FlatParam{
				Path: node.Path,
				Meta: node.Meta,
			},
		}
	}
	return m, nil
}

// updateDetailPanel handles key events when the detail panel is open.
func (m SSMTreeModel) updateDetailPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "v", "esc":
		// Close the detail panel and return to tree navigation.
		m.showDetail = false
		m.detailNode = nil
		m.detail = nil
		m.detailErr = nil
		m.decrypted = false
		m.detailLoading = false
		return m, nil
	case "d":
		// Decrypt SecureString value.
		if m.detailNode != nil && m.detailNode.IsSecureString() && !m.decrypted && m.fetchDetail != nil {
			m.detailLoading = true
			m.decrypted = true
			return m, m.fetchDetail(m.detailNode.Path, true)
		}
		return m, nil
	case "q", "ctrl+c":
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

// View implements tea.Model. It renders the tree with indentation, folder
// icons, and a cursor indicator. When the detail panel is open, it renders
// below the tree listing.
func (m SSMTreeModel) View() string {
	var sb strings.Builder

	if m.header != "" {
		sb.WriteString(Dim.Render(m.header) + "\n\n")
	}

	if m.root == nil || len(m.visible) == 0 {
		sb.WriteString(Yellow.Render("No parameters found") + "\n\n")
		sb.WriteString(Dim.Render("esc/q: back"))
		return sb.String()
	}

	for i, row := range m.visible {
		cursor := "  "
		if i == m.cursor {
			cursor = Cursor.Render("❯ ")
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
		if i == m.cursor {
			label = Selected.Render(label)
		}

		// Show type hint for parameter nodes.
		hint := ""
		if row.node.IsParameter() && row.node.Meta != nil {
			hint = " " + Dim.Render(fmt.Sprintf("[%s]", row.node.Meta.Type))
		}

		// Show child count for folders.
		if row.node.IsFolder() {
			count := row.node.ParameterCount()
			hint = " " + Dim.Render(fmt.Sprintf("(%d)", count))
		}

		// SecureString indicator.
		if row.node.IsSecureString() {
			hint += " " + Yellow.Render("🔒")
		}

		fmt.Fprintf(&sb, "%s%s%s%s%s\n", cursor, indent, icon, label, hint)
	}

	// Render detail panel if open.
	if m.showDetail {
		sb.WriteString("\n")
		sb.WriteString(m.renderDetailPanel())
	}

	// Help text changes depending on whether the detail or help panel is open.
	sb.WriteString("\n")
	if overlay := m.help.View(); overlay != "" {
		sb.WriteString(overlay)
	} else if m.showDetail {
		helpParts := []string{"v/esc: close detail"}
		if m.detailNode != nil && m.detailNode.IsSecureString() && !m.decrypted {
			helpParts = append(helpParts, "d: decrypt value")
		}
		helpParts = append(helpParts, "?: help", "q: quit")
		sb.WriteString(Dim.Render(strings.Join(helpParts, "  ")))
	} else {
		sb.WriteString(HelpBar(
			SSMTreeKeys.Move,
			SSMTreeKeys.Expand,
			SSMTreeKeys.Collapse,
			SSMTreeKeys.Toggle,
			SSMTreeKeys.Detail,
			SSMTreeKeys.Cancel,
			SSMTreeKeys.Help,
		))
	}

	return sb.String()
}

// renderDetailPanel renders the inline parameter detail panel.
func (m SSMTreeModel) renderDetailPanel() string {
	var sb strings.Builder

	// Panel border top.
	sb.WriteString(Dim.Render("─── Parameter Detail ───") + "\n")

	if m.detailLoading {
		sb.WriteString(Dim.Render("  Loading...") + "\n")
		return sb.String()
	}

	if m.detailErr != nil {
		sb.WriteString(Red.Render("  Error: "+m.detailErr.Error()) + "\n")
		return sb.String()
	}

	if m.detailNode == nil {
		return sb.String()
	}

	// Name and path.
	sb.WriteString(fmt.Sprintf("  %s  %s\n", Dim.Render("Name:"), Bold.Render(m.detailNode.Name)))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", Dim.Render("Path:"), m.detailNode.Path))

	// Type.
	if m.detailNode.Meta != nil {
		typeStr := m.detailNode.Meta.Type
		if m.detailNode.IsSecureString() {
			typeStr = Yellow.Render(typeStr + " 🔒")
		} else {
			typeStr = Cyan.Render(typeStr)
		}
		sb.WriteString(fmt.Sprintf("  %s  %s\n", Dim.Render("Type:"), typeStr))
	}

	// Value — from fetched detail if available.
	if m.detail != nil && m.detail.Value != "" {
		valueLabel := "Value:"
		if m.detailNode.IsSecureString() {
			if m.decrypted {
				sb.WriteString(fmt.Sprintf("  %s %s\n", Dim.Render(valueLabel), Yellow.Render(m.detail.Value)))
			} else {
				sb.WriteString(fmt.Sprintf("  %s %s  %s\n", Dim.Render(valueLabel), Dim.Render("••••••••"), Dim.Render("(press d to decrypt)")))
			}
		} else {
			sb.WriteString(fmt.Sprintf("  %s %s\n", Dim.Render(valueLabel), Green.Render(m.detail.Value)))
		}
	} else if m.detailNode.IsSecureString() {
		sb.WriteString(fmt.Sprintf("  %s %s  %s\n", Dim.Render("Value:"), Dim.Render("••••••••"), Dim.Render("(press d to decrypt)")))
	}

	// Metadata fields from node meta or fetched detail.
	meta := m.detailNode.Meta
	if m.detail != nil && m.detail.Meta != nil {
		meta = m.detail.Meta
	}

	if meta != nil {
		if meta.Version > 0 {
			sb.WriteString(fmt.Sprintf("  %s  %d\n", Dim.Render("Version:"), meta.Version))
		}
		if !meta.LastModified.IsZero() {
			sb.WriteString(fmt.Sprintf("  %s  %s\n", Dim.Render("Modified:"), meta.LastModified.Format("2006-01-02 15:04:05 UTC")))
		}
		if meta.DataType != "" {
			sb.WriteString(fmt.Sprintf("  %s  %s\n", Dim.Render("DataType:"), meta.DataType))
		}
		if meta.ARN != "" {
			sb.WriteString(fmt.Sprintf("  %s  %s\n", Dim.Render("ARN:"), Dim.Render(meta.ARN)))
		}
	}

	sb.WriteString(Dim.Render("────────────────────────"))

	return sb.String()
}

// ShowingDetail returns whether the detail panel is currently visible.
func (m SSMTreeModel) ShowingDetail() bool {
	return m.showDetail
}

// DetailInfo returns the current detail info being displayed, or nil.
func (m SSMTreeModel) DetailInfo() *ssm.ParameterDetail {
	return m.detail
}

// DetailError returns the error from the last detail fetch, or nil.
func (m SSMTreeModel) DetailError() error {
	return m.detailErr
}

// IsDecrypted returns whether the current detail has been decrypted.
func (m SSMTreeModel) IsDecrypted() bool {
	return m.decrypted
}

// SelectedNode returns the parameter node chosen by the user, or nil if cancelled.
func (m SSMTreeModel) SelectedNode() *ssm.TreeNode {
	if m.quit {
		return nil
	}
	return m.selected
}

// CursorNode returns the node currently under the cursor, or nil if empty.
func (m SSMTreeModel) CursorNode() *ssm.TreeNode {
	if len(m.visible) == 0 || m.cursor < 0 || m.cursor >= len(m.visible) {
		return nil
	}
	return m.visible[m.cursor].node
}

// rebuildVisible flattens the tree into the visible slice based on expansion state.
func (m *SSMTreeModel) rebuildVisible() {
	m.visible = m.visible[:0]
	if m.root == nil {
		return
	}
	// Walk children of root (root itself is "/" and not displayed as a row).
	for _, child := range m.root.Children {
		m.flattenNode(child, 0)
	}
}

// flattenNode recursively appends visible nodes at the given depth.
func (m *SSMTreeModel) flattenNode(node *ssm.TreeNode, depth int) {
	m.visible = append(m.visible, &visibleRow{node: node, depth: depth})
	if node.IsFolder() && node.Expanded {
		for _, child := range node.Children {
			m.flattenNode(child, depth+1)
		}
	}
}

// jumpToParent moves the cursor to the nearest ancestor folder at a lower depth.
func (m *SSMTreeModel) jumpToParent() {
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

// RunSSMTree runs the SSM tree TUI and returns the selected parameter node.
// Returns nil if the user cancelled. Renders to stderr so stdout stays clean.
func RunSSMTree(root *ssm.TreeNode, header string) (*ssm.TreeNode, error) {
	m := NewSSMTree(root, header)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("SSM tree browser error: %w", err)
	}
	return finalModel.(SSMTreeModel).SelectedNode(), nil
}
