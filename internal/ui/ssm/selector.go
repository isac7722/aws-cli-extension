package ssm

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/isac7722/aws-cli-extension/internal/config"
	"github.com/isac7722/aws-cli-extension/internal/tui"
)

// profilesLoadedMsg is sent after profiles are loaded from AWS config files.
type profilesLoadedMsg struct {
	profiles []config.Profile
	err      error
}

// ProfileSelectorModel is a Bubble Tea model for selecting an AWS profile
// before entering the SSM Parameter Store browser.
type ProfileSelectorModel struct {
	profiles []config.Profile
	cursor   int
	chosen   int
	loading  bool
	err      error
	quit     bool
	current  string // currently active profile (from AWS_PROFILE env)
}

// NewProfileSelector creates a new profile selector model.
// current is the name of the currently active AWS profile (may be empty).
func NewProfileSelector(current string) ProfileSelectorModel {
	return ProfileSelectorModel{
		loading: true,
		chosen:  -1,
		current: current,
	}
}

// loadProfiles is a tea.Cmd that loads AWS profiles from ~/.aws/credentials and ~/.aws/config.
func loadProfiles() tea.Msg {
	cfg, err := config.LoadProfiles()
	if err != nil {
		return profilesLoadedMsg{err: err}
	}
	return profilesLoadedMsg{profiles: cfg.Profiles}
}

func (m ProfileSelectorModel) Init() tea.Cmd {
	return loadProfiles
}

func (m ProfileSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case profilesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.profiles = msg.profiles
		if len(m.profiles) == 0 {
			m.err = fmt.Errorf("no AWS profiles found — run 'awse user add' to create one")
			return m, nil
		}
		// Position cursor on the currently active profile if present.
		for i, p := range m.profiles {
			if p.Name == m.current {
				m.cursor = i
				break
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.err != nil {
			m.quit = true
			return m, tea.Quit
		}
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.profiles)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.profiles) > 0 {
				m.chosen = m.cursor
				return m, tea.Quit
			}
		case "esc", "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ProfileSelectorModel) View() string {
	var sb strings.Builder

	sb.WriteString(tui.Dim.Render("Select AWS profile for SSM") + "\n\n")

	if m.loading {
		sb.WriteString(tui.Dim.Render("Loading profiles...") + "\n")
		return sb.String()
	}

	if m.err != nil {
		sb.WriteString(tui.Red.Render("Error: "+m.err.Error()) + "\n\n")
		sb.WriteString(tui.Dim.Render("Press any key to exit"))
		return sb.String()
	}

	for i, p := range m.profiles {
		cursor := "  "
		if i == m.cursor {
			cursor = tui.Cursor.Render("❯ ")
		}

		label := p.Name
		if i == m.cursor {
			label = tui.Selected.Render(label)
		}

		// Show region as a hint if available.
		hint := ""
		if p.Region != "" {
			hint = " " + tui.Dim.Render(p.Region)
		}

		// Show a check mark next to the currently active profile.
		marker := ""
		if p.Name == m.current {
			marker = " " + tui.Green.Render("✔")
		}

		// Show credential status indicator.
		status := ""
		if !p.HasCredentials() {
			status = " " + tui.Yellow.Render("(no credentials)")
		}

		fmt.Fprintf(&sb, "%s%s%s%s%s\n", cursor, label, hint, marker, status)
	}

	sb.WriteString("\n" + tui.Dim.Render("↑↓/jk: move  ⏎: select  esc/q: cancel"))

	return sb.String()
}

// Chosen returns the index of the selected profile, or -1 if cancelled.
func (m ProfileSelectorModel) Chosen() int {
	if m.quit {
		return -1
	}
	return m.chosen
}

// ChosenProfile returns the selected profile, or nil if cancelled.
func (m ProfileSelectorModel) ChosenProfile() *config.Profile {
	idx := m.Chosen()
	if idx < 0 || idx >= len(m.profiles) {
		return nil
	}
	p := m.profiles[idx]
	return &p
}

// RunProfileSelector runs the profile selector TUI and returns the chosen profile.
// Returns nil if the user cancelled. Renders to stderr so stdout stays clean for
// shell wrappers (eval $(awse ...)).
func RunProfileSelector(currentProfile string) (*config.Profile, error) {
	m := NewProfileSelector(currentProfile)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("profile selector error: %w", err)
	}
	return finalModel.(ProfileSelectorModel).ChosenProfile(), nil
}

// AWSRegions is the list of commonly available AWS regions.
var AWSRegions = []string{
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
	"af-south-1",
	"ap-east-1",
	"ap-south-1",
	"ap-south-2",
	"ap-southeast-1",
	"ap-southeast-2",
	"ap-southeast-3",
	"ap-northeast-1",
	"ap-northeast-2",
	"ap-northeast-3",
	"ca-central-1",
	"eu-central-1",
	"eu-central-2",
	"eu-west-1",
	"eu-west-2",
	"eu-west-3",
	"eu-south-1",
	"eu-south-2",
	"eu-north-1",
	"me-south-1",
	"me-central-1",
	"sa-east-1",
}

// regionDescriptions provides friendly names for AWS regions.
var regionDescriptions = map[string]string{
	"us-east-1":      "N. Virginia",
	"us-east-2":      "Ohio",
	"us-west-1":      "N. California",
	"us-west-2":      "Oregon",
	"af-south-1":     "Cape Town",
	"ap-east-1":      "Hong Kong",
	"ap-south-1":     "Mumbai",
	"ap-south-2":     "Hyderabad",
	"ap-southeast-1": "Singapore",
	"ap-southeast-2": "Sydney",
	"ap-southeast-3": "Jakarta",
	"ap-northeast-1": "Tokyo",
	"ap-northeast-2": "Seoul",
	"ap-northeast-3": "Osaka",
	"ca-central-1":   "Canada",
	"eu-central-1":   "Frankfurt",
	"eu-central-2":   "Zurich",
	"eu-west-1":      "Ireland",
	"eu-west-2":      "London",
	"eu-west-3":      "Paris",
	"eu-south-1":     "Milan",
	"eu-south-2":     "Spain",
	"eu-north-1":     "Stockholm",
	"me-south-1":     "Bahrain",
	"me-central-1":   "UAE",
	"sa-east-1":      "São Paulo",
}

// RegionSelectorModel is a Bubble Tea model for selecting an AWS region.
// It activates after a profile has been chosen in the SSM flow.
type RegionSelectorModel struct {
	regions []string
	cursor  int
	chosen  int
	quit    bool
	current string // pre-selected region (e.g., from profile config or AWS_REGION)
}

// NewRegionSelector creates a new region selector model.
// current is the region to pre-select (e.g., from the chosen profile or AWS_REGION env var).
// If current is empty, the cursor starts at the top.
func NewRegionSelector(current string) RegionSelectorModel {
	cursor := 0
	for i, r := range AWSRegions {
		if r == current {
			cursor = i
			break
		}
	}
	return RegionSelectorModel{
		regions: AWSRegions,
		cursor:  cursor,
		chosen:  -1,
		current: current,
	}
}

func (m RegionSelectorModel) Init() tea.Cmd { return nil }

func (m RegionSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.regions)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.regions) > 0 {
				m.chosen = m.cursor
				return m, tea.Quit
			}
		case "esc", "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m RegionSelectorModel) View() string {
	var sb strings.Builder

	sb.WriteString(tui.Dim.Render("Select AWS region") + "\n\n")

	for i, region := range m.regions {
		cursor := "  "
		if i == m.cursor {
			cursor = tui.Cursor.Render("❯ ")
		}

		label := region
		if i == m.cursor {
			label = tui.Selected.Render(label)
		}

		// Show region description as a hint.
		hint := ""
		if desc, ok := regionDescriptions[region]; ok {
			hint = " " + tui.Dim.Render(desc)
		}

		// Show a check mark next to the currently active region.
		marker := ""
		if region == m.current {
			marker = " " + tui.Green.Render("✔")
		}

		fmt.Fprintf(&sb, "%s%s%s%s\n", cursor, label, hint, marker)
	}

	sb.WriteString("\n" + tui.Dim.Render("↑↓/jk: move  ⏎: select  esc/q: cancel"))

	return sb.String()
}

// Chosen returns the index of the selected region, or -1 if cancelled.
func (m RegionSelectorModel) Chosen() int {
	if m.quit {
		return -1
	}
	return m.chosen
}

// ChosenRegion returns the selected region string, or "" if cancelled.
func (m RegionSelectorModel) ChosenRegion() string {
	idx := m.Chosen()
	if idx < 0 || idx >= len(m.regions) {
		return ""
	}
	return m.regions[idx]
}

// RunRegionSelector runs the region selector TUI and returns the chosen region.
// Returns "" if the user cancelled. Renders to stderr so stdout stays clean for
// shell wrappers (eval $(awse ...)).
func RunRegionSelector(currentRegion string) (string, error) {
	m := NewRegionSelector(currentRegion)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("region selector error: %w", err)
	}
	return finalModel.(RegionSelectorModel).ChosenRegion(), nil
}
