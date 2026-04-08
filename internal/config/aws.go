package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Profile represents an AWS credential profile merged from credentials and config files.
type Profile struct {
	Name            string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
	Output          string
}

// AWSConfig holds all loaded profiles from ~/.aws/credentials and ~/.aws/config.
type AWSConfig struct {
	Profiles []Profile
	byIdx    map[string]int // profile name → index in Profiles slice
}

// CredentialsPath returns the default AWS credentials file path.
func CredentialsPath() string {
	if p := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "credentials")
}

// ConfigPath returns the default AWS config file path.
func ConfigPath() string {
	if p := os.Getenv("AWS_CONFIG_FILE"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "config")
}

// LoadProfiles reads profiles from both ~/.aws/credentials and ~/.aws/config.
func LoadProfiles() (*AWSConfig, error) {
	cfg := &AWSConfig{byIdx: make(map[string]int)}

	// Load credentials file (has access keys)
	if err := cfg.loadCredentials(CredentialsPath()); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	// Load config file (has regions)
	if err := cfg.loadConfig(ConfigPath()); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return cfg, nil
}

func (c *AWSConfig) loadCredentials(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	var current *Profile

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(line[1 : len(line)-1])
			current = c.getOrCreate(name)
			continue
		}

		if current != nil {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			switch key {
			case "aws_access_key_id":
				current.AccessKeyID = val
			case "aws_secret_access_key":
				current.SecretAccessKey = val
			case "aws_session_token":
				current.SessionToken = val
			case "region":
				current.Region = val
			case "output":
				current.Output = val
			}
		}
	}

	return scanner.Err()
}

func (c *AWSConfig) loadConfig(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	var current *Profile

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.TrimSpace(line[1 : len(line)-1])
			// AWS config uses "profile xxx" prefix for non-default profiles
			name := section
			if strings.HasPrefix(section, "profile ") {
				name = strings.TrimPrefix(section, "profile ")
			}
			current = c.getOrCreate(name)
			continue
		}

		if current != nil {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			switch key {
			case "region":
				// Credentials file region takes priority; config is fallback
				if current.Region == "" {
					current.Region = val
				}
			case "output":
				if current.Output == "" {
					current.Output = val
				}
			}
		}
	}

	return scanner.Err()
}

func (c *AWSConfig) getOrCreate(name string) *Profile {
	if idx, ok := c.byIdx[name]; ok {
		return &c.Profiles[idx]
	}
	c.Profiles = append(c.Profiles, Profile{Name: name})
	idx := len(c.Profiles) - 1
	c.byIdx[name] = idx
	return &c.Profiles[idx]
}

// Get returns a profile by name.
func (c *AWSConfig) Get(name string) (*Profile, bool) {
	if idx, ok := c.byIdx[name]; ok {
		return &c.Profiles[idx], true
	}
	return nil, false
}

// iniSection represents a parsed section from an INI file, preserving raw lines.
type iniSection struct {
	// header is the full section header line, e.g. "[default]" or "[profile production]".
	header string
	// name is the resolved profile name (e.g. "production" from "[profile production]").
	name string
	// lines holds every non-header line in the section (key=value, comments, blanks).
	lines []string
}

// parseINI reads an INI file and returns its sections plus any preamble (lines before
// the first section header). Comments, blank lines, and unknown keys are preserved.
func parseINI(path string) (preamble []string, sections []iniSection, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var current *iniSection

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			// Start a new section
			if current != nil {
				sections = append(sections, *current)
			}
			sectionContent := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			name := sectionContent
			if strings.HasPrefix(sectionContent, "profile ") {
				name = strings.TrimPrefix(sectionContent, "profile ")
			}
			current = &iniSection{
				header: line,
				name:   name,
			}
			continue
		}

		if current == nil {
			preamble = append(preamble, line)
		} else {
			current.lines = append(current.lines, line)
		}
	}
	if current != nil {
		sections = append(sections, *current)
	}

	return preamble, sections, scanner.Err()
}

// setINIKey updates or appends a key=value pair in a section's lines.
// Returns the updated lines slice.
func setINIKey(lines []string, key, value string) []string {
	prefix := key + " "
	prefixEq := key + "="
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) || strings.HasPrefix(trimmed, prefixEq) {
			lines[i] = fmt.Sprintf("%s = %s", key, value)
			return lines
		}
	}
	// Append the new key at the end (before any trailing blank lines)
	insertIdx := len(lines)
	for insertIdx > 0 && strings.TrimSpace(lines[insertIdx-1]) == "" {
		insertIdx--
	}
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:insertIdx]...)
	newLines = append(newLines, fmt.Sprintf("%s = %s", key, value))
	newLines = append(newLines, lines[insertIdx:]...)
	return newLines
}

// removeINIKey removes a key from a section's lines if present.
func removeINIKey(lines []string, key string) []string {
	prefix := key + " "
	prefixEq := key + "="
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) || strings.HasPrefix(trimmed, prefixEq) {
			continue
		}
		result = append(result, line)
	}
	return result
}

// profileByName returns the index of a profile matching name, or -1 if not found.
func (c *AWSConfig) profileByName(name string) int {
	for i, p := range c.Profiles {
		if p.Name == name {
			return i
		}
	}
	return -1
}

// Save writes credentials and config back to their respective files.
// It uses a patch approach: existing file contents are read first, then only
// the known profile fields are updated. Comments, unknown keys, and formatting
// in other profiles are preserved.
func (c *AWSConfig) Save() error {
	if err := c.saveCredentials(CredentialsPath()); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}
	if err := c.saveConfig(ConfigPath()); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

func (c *AWSConfig) saveCredentials(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	preamble, sections, err := parseINI(path)
	if err != nil {
		return err
	}

	// Build a set of current profile names for tracking which are new/removed.
	profileSet := make(map[string]bool, len(c.Profiles))
	for _, p := range c.Profiles {
		profileSet[p.Name] = true
	}

	// Update existing sections.
	var updatedSections []iniSection
	seen := make(map[string]bool)
	for _, sec := range sections {
		if !profileSet[sec.name] {
			// Profile was deleted — skip this section.
			continue
		}
		seen[sec.name] = true
		idx := c.profileByName(sec.name)
		if idx < 0 {
			continue
		}
		p := c.Profiles[idx]
		sec.lines = c.patchCredentialLines(sec.lines, p)
		updatedSections = append(updatedSections, sec)
	}

	// Append new profiles that weren't in the original file.
	for _, p := range c.Profiles {
		if seen[p.Name] {
			continue
		}
		// Only write to credentials file if the profile has credentials.
		if !p.HasCredentials() && p.SessionToken == "" {
			continue
		}
		sec := iniSection{
			header: fmt.Sprintf("[%s]", p.Name),
			name:   p.Name,
		}
		sec.lines = c.patchCredentialLines(nil, p)
		updatedSections = append(updatedSections, sec)
	}

	return writeINI(path, preamble, updatedSections)
}

// patchCredentialLines updates credential-related keys in lines for the given profile.
func (c *AWSConfig) patchCredentialLines(lines []string, p Profile) []string {
	type keyVal struct {
		key string
		val string
	}
	fields := []keyVal{
		{"aws_access_key_id", p.AccessKeyID},
		{"aws_secret_access_key", p.SecretAccessKey},
		{"aws_session_token", p.SessionToken},
	}

	for _, f := range fields {
		if f.val != "" {
			lines = setINIKey(lines, f.key, f.val)
		} else {
			lines = removeINIKey(lines, f.key)
		}
	}
	return lines
}

func (c *AWSConfig) saveConfig(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	preamble, sections, err := parseINI(path)
	if err != nil {
		return err
	}

	profileSet := make(map[string]bool, len(c.Profiles))
	for _, p := range c.Profiles {
		profileSet[p.Name] = true
	}

	var updatedSections []iniSection
	seen := make(map[string]bool)
	for _, sec := range sections {
		if !profileSet[sec.name] {
			continue
		}
		seen[sec.name] = true
		idx := c.profileByName(sec.name)
		if idx < 0 {
			continue
		}
		p := c.Profiles[idx]
		sec.lines = c.patchConfigLines(sec.lines, p)
		updatedSections = append(updatedSections, sec)
	}

	// Append new profiles not in original config file.
	for _, p := range c.Profiles {
		if seen[p.Name] {
			continue
		}
		// Only write config entry if there's region or output to store.
		if p.Region == "" && p.Output == "" {
			continue
		}
		header := fmt.Sprintf("[profile %s]", p.Name)
		if p.Name == "default" {
			header = "[default]"
		}
		sec := iniSection{
			header: header,
			name:   p.Name,
		}
		sec.lines = c.patchConfigLines(nil, p)
		updatedSections = append(updatedSections, sec)
	}

	return writeINI(path, preamble, updatedSections)
}

// patchConfigLines updates config-related keys in lines for the given profile.
func (c *AWSConfig) patchConfigLines(lines []string, p Profile) []string {
	type keyVal struct {
		key string
		val string
	}
	fields := []keyVal{
		{"region", p.Region},
		{"output", p.Output},
	}

	for _, f := range fields {
		if f.val != "" {
			lines = setINIKey(lines, f.key, f.val)
		} else {
			lines = removeINIKey(lines, f.key)
		}
	}
	return lines
}

// writeINI writes preamble lines and sections to the given path with 0600 permissions.
func writeINI(path string, preamble []string, sections []iniSection) error {
	var sb strings.Builder

	for _, line := range preamble {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	for i, sec := range sections {
		if i > 0 || len(preamble) > 0 {
			// Ensure a blank line separator between sections (and after preamble).
			// But avoid double-blanks if already present.
			current := sb.String()
			if len(current) > 0 && !strings.HasSuffix(current, "\n\n") {
				sb.WriteString("\n")
			}
		}
		sb.WriteString(sec.header)
		sb.WriteString("\n")
		for _, line := range sec.lines {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0600)
}

// ProfileNames returns a sorted list of all profile names.
func (c *AWSConfig) ProfileNames() []string {
	names := make([]string, len(c.Profiles))
	for i, p := range c.Profiles {
		names[i] = p.Name
	}
	return names
}

// HasCredentials returns true if the profile has access key credentials.
func (p *Profile) HasCredentials() bool {
	return p.AccessKeyID != "" && p.SecretAccessKey != ""
}

// UpdateProfile updates an existing profile by name. Returns true if found.
// If updated.Name differs from name, the profile is renamed.
func (c *AWSConfig) UpdateProfile(name string, updated Profile) bool {
	if idx, ok := c.byIdx[name]; ok {
		c.Profiles[idx] = updated
		if name != updated.Name {
			delete(c.byIdx, name)
			c.byIdx[updated.Name] = idx
		}
		return true
	}
	return false
}

// AddProfile adds a new profile.
func (c *AWSConfig) AddProfile(p Profile) {
	if c.byIdx == nil {
		c.byIdx = make(map[string]int)
	}
	c.Profiles = append(c.Profiles, p)
	c.byIdx[p.Name] = len(c.Profiles) - 1
}

// RemoveProfile removes a profile by name. Returns true if found.
func (c *AWSConfig) RemoveProfile(name string) bool {
	idx, ok := c.byIdx[name]
	if !ok {
		return false
	}
	c.Profiles = append(c.Profiles[:idx], c.Profiles[idx+1:]...)
	delete(c.byIdx, name)
	// Rebuild index for shifted elements
	for i := idx; i < len(c.Profiles); i++ {
		c.byIdx[c.Profiles[i].Name] = i
	}
	return true
}

// MaskKey returns a masked version of an access key (shows first 4 and last 4 chars).
func MaskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
