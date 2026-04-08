package ssm

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/isac7722/aws-cli-extension/internal/config"
)

func TestNewProfileSelector_InitialState(t *testing.T) {
	m := NewProfileSelector("production")

	if !m.loading {
		t.Error("expected loading to be true initially")
	}
	if m.chosen != -1 {
		t.Errorf("expected chosen to be -1, got %d", m.chosen)
	}
	if m.current != "production" {
		t.Errorf("expected current to be 'production', got %q", m.current)
	}
	if m.quit {
		t.Error("expected quit to be false initially")
	}
}

func TestProfileSelectorModel_ProfilesLoaded(t *testing.T) {
	m := NewProfileSelector("staging")
	m.loading = true

	profiles := []config.Profile{
		{Name: "default", Region: "us-east-1", AccessKeyID: "AKIA1", SecretAccessKey: "secret1"},
		{Name: "staging", Region: "eu-west-1", AccessKeyID: "AKIA2", SecretAccessKey: "secret2"},
		{Name: "production", Region: "ap-northeast-2", AccessKeyID: "AKIA3", SecretAccessKey: "secret3"},
	}

	updated, _ := m.Update(profilesLoadedMsg{profiles: profiles})
	m = updated.(ProfileSelectorModel)

	if m.loading {
		t.Error("expected loading to be false after profilesLoadedMsg")
	}
	if len(m.profiles) != 3 {
		t.Errorf("expected 3 profiles, got %d", len(m.profiles))
	}
	// Cursor should be on "staging" (index 1) since it's the current profile.
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 (staging), got %d", m.cursor)
	}
}

func TestProfileSelectorModel_ProfilesLoadedError(t *testing.T) {
	m := NewProfileSelector("")
	m.loading = true

	updated, _ := m.Update(profilesLoadedMsg{err: errTest})
	m = updated.(ProfileSelectorModel)

	if m.loading {
		t.Error("expected loading to be false")
	}
	if m.err == nil {
		t.Error("expected error to be set")
	}
}

var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }

func TestProfileSelectorModel_NavigationKeys(t *testing.T) {
	m := profileSelectorWithProfiles(3, "")

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(ProfileSelectorModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after j, got %d", m.cursor)
	}

	// Move down again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(ProfileSelectorModel)
	if m.cursor != 2 {
		t.Errorf("expected cursor 2 after j, got %d", m.cursor)
	}

	// Move down at bottom (should stay)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(ProfileSelectorModel)
	if m.cursor != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", m.cursor)
	}

	// Move up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(ProfileSelectorModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after k, got %d", m.cursor)
	}
}

func TestProfileSelectorModel_Enter(t *testing.T) {
	m := profileSelectorWithProfiles(3, "")
	// Move to index 1 then press enter
	m.cursor = 1

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(ProfileSelectorModel)

	if m.chosen != 1 {
		t.Errorf("expected chosen 1, got %d", m.chosen)
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestProfileSelectorModel_Escape(t *testing.T) {
	m := profileSelectorWithProfiles(3, "")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(ProfileSelectorModel)

	if !m.quit {
		t.Error("expected quit to be true")
	}
	if m.Chosen() != -1 {
		t.Errorf("expected Chosen() to return -1 on cancel, got %d", m.Chosen())
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestProfileSelectorModel_ChosenProfile(t *testing.T) {
	m := profileSelectorWithProfiles(3, "")
	m.chosen = 1

	p := m.ChosenProfile()
	if p == nil {
		t.Fatal("expected non-nil profile")
	}
	if p.Name != "profile-1" {
		t.Errorf("expected profile-1, got %s", p.Name)
	}
}

func TestProfileSelectorModel_ChosenProfileNilOnCancel(t *testing.T) {
	m := profileSelectorWithProfiles(3, "")
	m.quit = true

	p := m.ChosenProfile()
	if p != nil {
		t.Error("expected nil profile on cancel")
	}
}

func TestProfileSelectorModel_ViewContainsProfiles(t *testing.T) {
	m := profileSelectorWithProfiles(3, "profile-0")

	view := m.View()

	if !containsString(view, "profile-0") {
		t.Error("expected view to contain profile-0")
	}
	if !containsString(view, "profile-1") {
		t.Error("expected view to contain profile-1")
	}
	if !containsString(view, "Select AWS profile") {
		t.Error("expected view to contain header")
	}
}

func TestProfileSelectorModel_ViewShowsLoading(t *testing.T) {
	m := NewProfileSelector("")

	view := m.View()

	if !containsString(view, "Loading") {
		t.Error("expected view to show loading state")
	}
}

func TestProfileSelectorModel_ViewShowsError(t *testing.T) {
	m := NewProfileSelector("")
	m.loading = false
	m.err = errTest

	view := m.View()

	if !containsString(view, "Error") {
		t.Error("expected view to show error")
	}
}

func TestProfileSelectorModel_EmptyProfilesError(t *testing.T) {
	m := NewProfileSelector("")
	m.loading = true

	updated, _ := m.Update(profilesLoadedMsg{profiles: nil})
	m = updated.(ProfileSelectorModel)

	if m.err == nil {
		t.Error("expected error for empty profiles")
	}
}

// helpers

func profileSelectorWithProfiles(n int, current string) ProfileSelectorModel {
	m := NewProfileSelector(current)
	m.loading = false
	m.profiles = make([]config.Profile, n)
	for i := 0; i < n; i++ {
		m.profiles[i] = config.Profile{
			Name:            fmt.Sprintf("profile-%d", i),
			Region:          "us-east-1",
			AccessKeyID:     "AKIATEST",
			SecretAccessKey: "secret",
		}
	}
	if current != "" {
		for i, p := range m.profiles {
			if p.Name == current {
				m.cursor = i
				break
			}
		}
	}
	return m
}

// Region selector tests

func TestNewRegionSelector_InitialState(t *testing.T) {
	m := NewRegionSelector("")

	if m.chosen != -1 {
		t.Errorf("expected chosen to be -1, got %d", m.chosen)
	}
	if m.quit {
		t.Error("expected quit to be false initially")
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0 for empty current, got %d", m.cursor)
	}
	if len(m.regions) != len(AWSRegions) {
		t.Errorf("expected %d regions, got %d", len(AWSRegions), len(m.regions))
	}
}

func TestNewRegionSelector_PreSelectsCurrent(t *testing.T) {
	m := NewRegionSelector("ap-northeast-2")

	expectedIdx := -1
	for i, r := range AWSRegions {
		if r == "ap-northeast-2" {
			expectedIdx = i
			break
		}
	}

	if m.cursor != expectedIdx {
		t.Errorf("expected cursor at %d (ap-northeast-2), got %d", expectedIdx, m.cursor)
	}
	if m.current != "ap-northeast-2" {
		t.Errorf("expected current to be 'ap-northeast-2', got %q", m.current)
	}
}

func TestNewRegionSelector_UnknownRegionStartsAtTop(t *testing.T) {
	m := NewRegionSelector("xx-unknown-1")

	if m.cursor != 0 {
		t.Errorf("expected cursor at 0 for unknown region, got %d", m.cursor)
	}
}

func TestRegionSelectorModel_NavigationKeys(t *testing.T) {
	m := NewRegionSelector("")

	// Move down with j
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(RegionSelectorModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after j, got %d", m.cursor)
	}

	// Move down again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(RegionSelectorModel)
	if m.cursor != 2 {
		t.Errorf("expected cursor 2 after j, got %d", m.cursor)
	}

	// Move up with k
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(RegionSelectorModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after k, got %d", m.cursor)
	}

	// Move up at top (should stay)
	m.cursor = 0
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(RegionSelectorModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", m.cursor)
	}
}

func TestRegionSelectorModel_NavigateToBottom(t *testing.T) {
	m := NewRegionSelector("")
	m.cursor = len(AWSRegions) - 1

	// Move down at bottom (should stay)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(RegionSelectorModel)
	if m.cursor != len(AWSRegions)-1 {
		t.Errorf("expected cursor to stay at bottom %d, got %d", len(AWSRegions)-1, m.cursor)
	}
}

func TestRegionSelectorModel_Enter(t *testing.T) {
	m := NewRegionSelector("")
	m.cursor = 3 // us-west-2

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(RegionSelectorModel)

	if m.chosen != 3 {
		t.Errorf("expected chosen 3, got %d", m.chosen)
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
	if m.ChosenRegion() != "us-west-2" {
		t.Errorf("expected ChosenRegion() 'us-west-2', got %q", m.ChosenRegion())
	}
}

func TestRegionSelectorModel_Escape(t *testing.T) {
	m := NewRegionSelector("")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(RegionSelectorModel)

	if !m.quit {
		t.Error("expected quit to be true")
	}
	if m.Chosen() != -1 {
		t.Errorf("expected Chosen() to return -1 on cancel, got %d", m.Chosen())
	}
	if m.ChosenRegion() != "" {
		t.Errorf("expected ChosenRegion() to be empty on cancel, got %q", m.ChosenRegion())
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestRegionSelectorModel_QuitKey(t *testing.T) {
	m := NewRegionSelector("")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(RegionSelectorModel)

	if !m.quit {
		t.Error("expected quit to be true on q")
	}
	if cmd == nil {
		t.Error("expected quit command on q")
	}
}

func TestRegionSelectorModel_CtrlC(t *testing.T) {
	m := NewRegionSelector("")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(RegionSelectorModel)

	if !m.quit {
		t.Error("expected quit to be true on ctrl+c")
	}
	if cmd == nil {
		t.Error("expected quit command on ctrl+c")
	}
}

func TestRegionSelectorModel_ChosenRegion(t *testing.T) {
	m := NewRegionSelector("")
	m.chosen = 0

	region := m.ChosenRegion()
	if region != "us-east-1" {
		t.Errorf("expected 'us-east-1', got %q", region)
	}
}

func TestRegionSelectorModel_ChosenRegionOutOfBounds(t *testing.T) {
	m := NewRegionSelector("")
	m.chosen = 999

	region := m.ChosenRegion()
	if region != "" {
		t.Errorf("expected empty string for out of bounds, got %q", region)
	}
}

func TestRegionSelectorModel_ViewContainsRegions(t *testing.T) {
	m := NewRegionSelector("us-east-1")

	view := m.View()

	if !containsString(view, "us-east-1") {
		t.Error("expected view to contain us-east-1")
	}
	if !containsString(view, "us-west-2") {
		t.Error("expected view to contain us-west-2")
	}
	if !containsString(view, "Select AWS region") {
		t.Error("expected view to contain header")
	}
}

func TestRegionSelectorModel_ViewShowsDescriptions(t *testing.T) {
	m := NewRegionSelector("")

	view := m.View()

	if !containsString(view, "N. Virginia") {
		t.Error("expected view to contain region description 'N. Virginia'")
	}
	if !containsString(view, "Seoul") {
		t.Error("expected view to contain region description 'Seoul'")
	}
}

func TestRegionSelectorModel_ViewShowsCurrentMarker(t *testing.T) {
	m := NewRegionSelector("eu-west-1")

	view := m.View()

	// The view should contain the ✔ marker somewhere
	if !containsString(view, "✔") {
		t.Error("expected view to show ✔ marker for current region")
	}
}

func TestRegionSelectorModel_ViewShowsHelpText(t *testing.T) {
	m := NewRegionSelector("")

	view := m.View()

	if !containsString(view, "move") {
		t.Error("expected view to contain navigation help")
	}
	if !containsString(view, "select") {
		t.Error("expected view to contain select help")
	}
	if !containsString(view, "cancel") {
		t.Error("expected view to contain cancel help")
	}
}

func TestRegionSelectorModel_Init(t *testing.T) {
	m := NewRegionSelector("")
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init() to return nil cmd")
	}
}

func TestAWSRegions_NotEmpty(t *testing.T) {
	if len(AWSRegions) == 0 {
		t.Error("AWSRegions should not be empty")
	}
}

func TestRegionDescriptions_CoverAllRegions(t *testing.T) {
	for _, region := range AWSRegions {
		if _, ok := regionDescriptions[region]; !ok {
			t.Errorf("missing description for region %q", region)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
