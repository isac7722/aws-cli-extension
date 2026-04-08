package tui

import (
	"testing"
)

func TestCommonKeys_HelpText(t *testing.T) {
	tests := []struct {
		name     string
		helpKey  string
		helpDesc string
	}{
		{"Quit", CommonKeys.Quit.Help().Key, "quit"},
		{"Escape", CommonKeys.Escape.Help().Key, "cancel"},
		{"Enter", CommonKeys.Enter.Help().Key, "confirm/select"},
		{"Up", CommonKeys.Up.Help().Key, "move up"},
		{"Down", CommonKeys.Down.Help().Key, "move down"},
		{"Help", CommonKeys.Help.Help().Key, "toggle help"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.helpKey == "" {
				t.Errorf("%s: help key is empty", tt.name)
			}
			if tt.helpDesc == "" {
				t.Errorf("%s: help desc is empty", tt.name)
			}
		})
	}
}

func TestSelectorKeys_HelpText(t *testing.T) {
	if SelectorKeys.Move.Help().Key == "" {
		t.Error("SelectorKeys.Move help key is empty")
	}
	if SelectorKeys.Select.Help().Key == "" {
		t.Error("SelectorKeys.Select help key is empty")
	}
	if SelectorKeys.Cancel.Help().Key == "" {
		t.Error("SelectorKeys.Cancel help key is empty")
	}
	if SelectorKeys.Help.Help().Key == "" {
		t.Error("SelectorKeys.Help help key is empty")
	}
}

func TestSSMTreeKeys_HelpText(t *testing.T) {
	keys := []struct {
		name string
		key  string
	}{
		{"Move", SSMTreeKeys.Move.Help().Key},
		{"Expand", SSMTreeKeys.Expand.Help().Key},
		{"Collapse", SSMTreeKeys.Collapse.Help().Key},
		{"Toggle", SSMTreeKeys.Toggle.Help().Key},
		{"Detail", SSMTreeKeys.Detail.Help().Key},
		{"Cancel", SSMTreeKeys.Cancel.Help().Key},
		{"Help", SSMTreeKeys.Help.Help().Key},
	}

	for _, k := range keys {
		if k.key == "" {
			t.Errorf("SSMTreeKeys.%s help key is empty", k.name)
		}
	}
}

func TestSSMBrowserKeys_HelpText(t *testing.T) {
	if SSMBrowserKeys.ShowValue.Help().Desc == "" {
		t.Error("SSMBrowserKeys.ShowValue help desc is empty")
	}
}

func TestProfileEditKeys_HelpText(t *testing.T) {
	if ProfileEditKeys.Next.Help().Key == "" {
		t.Error("ProfileEditKeys.Next help key is empty")
	}
	if ProfileEditKeys.Prev.Help().Key == "" {
		t.Error("ProfileEditKeys.Prev help key is empty")
	}
}

func TestSSMCreateKeys_HelpText(t *testing.T) {
	if SSMCreateKeys.ChangeType.Help().Desc == "" {
		t.Error("SSMCreateKeys.ChangeType help desc is empty")
	}
	if SSMCreateKeys.Submit.Help().Key == "" {
		t.Error("SSMCreateKeys.Submit help key is empty")
	}
}
