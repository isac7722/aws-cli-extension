package tui

import "github.com/charmbracelet/bubbles/key"

// KeyBinding pairs a key.Binding with a short description for the help overlay.
type KeyBinding struct {
	Binding key.Binding
	Help    string // short description shown in the help overlay
}

// CommonKeys defines key bindings shared across all TUI models.
var CommonKeys = struct {
	Quit   key.Binding
	Escape key.Binding
	Enter  key.Binding
	Up     key.Binding
	Down   key.Binding
	Help   key.Binding
}{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("⏎", "confirm/select"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
}

// ConfirmKeys defines key bindings specific to the confirmation prompt.
var ConfirmKeys = struct {
	Confirm key.Binding
	Deny    key.Binding
	Toggle  key.Binding
	Cancel  key.Binding
}{
	Confirm: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "yes"),
	),
	Deny: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "no"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("left", "right", "tab"),
		key.WithHelp("←→/tab", "toggle"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc/q", "cancel"),
	),
}

// SelectorKeys defines key bindings specific to the selector model.
var SelectorKeys = struct {
	Move   key.Binding
	Select key.Binding
	Cancel key.Binding
	Help   key.Binding
}{
	Move: key.NewBinding(
		key.WithKeys("up", "down", "k", "j"),
		key.WithHelp("↑↓/jk", "move"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("⏎", "select"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc/q", "cancel"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// SSMTreeKeys defines key bindings specific to the SSM tree browser.
var SSMTreeKeys = struct {
	Move     key.Binding
	Expand   key.Binding
	Collapse key.Binding
	Toggle   key.Binding
	Detail   key.Binding
	Cancel   key.Binding
	Help     key.Binding
}{
	Move: key.NewBinding(
		key.WithKeys("up", "down", "k", "j"),
		key.WithHelp("↑↓/jk", "move"),
	),
	Expand: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "expand"),
	),
	Collapse: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "collapse"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("⏎/space", "toggle/select"),
	),
	Detail: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "detail"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc/q", "cancel"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// SSMBrowserKeys defines key bindings specific to the SSM browser.
var SSMBrowserKeys = struct {
	Move      key.Binding
	Select    key.Binding
	ShowValue key.Binding
	Cancel    key.Binding
	Help      key.Binding
}{
	Move: key.NewBinding(
		key.WithKeys("up", "down", "k", "j"),
		key.WithHelp("↑↓/jk", "move"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("⏎", "select"),
	),
	ShowValue: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "show/hide value"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc/q", "cancel"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// ProfileEditKeys defines key bindings specific to the profile edit form.
var ProfileEditKeys = struct {
	Next   key.Binding
	Prev   key.Binding
	Submit key.Binding
	Cancel key.Binding
	Help   key.Binding
}{
	Next: key.NewBinding(
		key.WithKeys("tab", "down"),
		key.WithHelp("tab/↓", "next field"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab", "up"),
		key.WithHelp("shift+tab/↑", "prev field"),
	),
	Submit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("⏎", "next/submit"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// SSMUpdateKeys defines key bindings specific to the SSM update form.
var SSMUpdateKeys = struct {
	Next       key.Binding
	Prev       key.Binding
	ChangeType key.Binding
	Toggle     key.Binding
	Submit     key.Binding
	Cancel     key.Binding
	Help       key.Binding
}{
	Next: key.NewBinding(
		key.WithKeys("tab", "down"),
		key.WithHelp("tab/↓", "next field"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab", "up"),
		key.WithHelp("shift+tab/↑", "prev field"),
	),
	ChangeType: key.NewBinding(
		key.WithKeys("left", "right"),
		key.WithHelp("←→", "change type"),
	),
	Toggle: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
	Submit: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "submit"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// SSMCreateKeys defines key bindings specific to the SSM create form.
var SSMCreateKeys = struct {
	Next       key.Binding
	Prev       key.Binding
	ChangeType key.Binding
	Submit     key.Binding
	Cancel     key.Binding
	Help       key.Binding
}{
	Next: key.NewBinding(
		key.WithKeys("tab", "down"),
		key.WithHelp("tab/↓", "next field"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab", "up"),
		key.WithHelp("shift+tab/↑", "prev field"),
	),
	ChangeType: key.NewBinding(
		key.WithKeys("left", "right"),
		key.WithHelp("←→", "change type"),
	),
	Submit: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "submit"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}
