package uninstall

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/isac7722/aws-cli-extension/internal/shell"
)

// Target represents a single cleanup item.
type Target struct {
	Category    string // "shell"
	Description string
	Path        string
}

// Plan holds all discovered cleanup targets.
type Plan struct {
	Targets    []Target
	IsHomebrew bool
	BinaryPath string
}

// Discover scans the system for awse-related artifacts.
func Discover() *Plan {
	plan := &Plan{}

	// 1. Binary
	if binPath, err := exec.LookPath("awse"); err == nil {
		plan.BinaryPath = binPath
		plan.IsHomebrew = strings.Contains(binPath, "Cellar") || strings.Contains(binPath, "homebrew")
	}

	// 2. Shell RC files
	for _, rc := range shell.RCFiles() {
		if shell.HasMarker(rc) {
			plan.Targets = append(plan.Targets, Target{
				Category:    "shell",
				Description: fmt.Sprintf("%s — shell integration block", rc),
				Path:        rc,
			})
		}
	}

	return plan
}

// FormatPlan returns a human-readable preview of the plan.
func FormatPlan(plan *Plan) string {
	if len(plan.Targets) == 0 && plan.BinaryPath == "" {
		return "Nothing to uninstall."
	}

	var sb strings.Builder
	sb.WriteString("The following will be removed:\n")

	if len(plan.Targets) > 0 {
		sb.WriteString("\n  Shell integration:\n")
		for _, t := range plan.Targets {
			fmt.Fprintf(&sb, "    • %s\n", t.Description)
		}
	}

	if plan.BinaryPath != "" {
		fmt.Fprintf(&sb, "\n  Binary:\n    • %s (manual removal required)\n", plan.BinaryPath)
	}

	return sb.String()
}

// Execute removes all targets in the plan. Returns result messages.
func Execute(plan *Plan) []string {
	var results []string

	for _, t := range plan.Targets {
		if t.Category == "shell" {
			if err := shell.RemoveMarker(t.Path); err != nil {
				results = append(results, fmt.Sprintf("✗ Failed to remove shell block from %s: %v", t.Path, err))
			} else {
				results = append(results, fmt.Sprintf("✔ Removed shell block from %s", t.Path))
			}
		}
	}

	return results
}
