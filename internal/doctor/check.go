// Package doctor provides health-check utilities for verifying
// external tool availability (e.g. AWS CLI).
package doctor

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Status represents the outcome of a single check.
type Status int

const (
	StatusOK      Status = iota // tool found and meets requirements
	StatusWarning               // tool found but may have issues
	StatusError                 // tool not found or unusable
)

// String returns a human-readable label for the status.
func (s Status) String() string {
	switch s {
	case StatusOK:
		return "ok"
	case StatusWarning:
		return "warning"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// CheckResult holds the structured output of a single doctor check.
type CheckResult struct {
	Name    string // short identifier, e.g. "aws-cli"
	Status  Status
	Version string // parsed version string, empty when not found
	Message string // human-readable detail
}

// versionRe captures AWS CLI version strings like "aws-cli/2.15.30".
var versionRe = regexp.MustCompile(`aws-cli/(\d+\.\d+\.\d+)`)

// commandRunner abstracts exec.Command for testability.
type commandRunner func(name string, args ...string) ([]byte, error)

// defaultRunner uses the real os/exec path.
func defaultRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// CheckAWSCLI detects whether the AWS CLI is installed and returns a
// structured result including version information.
func CheckAWSCLI() CheckResult {
	return checkAWSCLIWith(defaultRunner)
}

// checkAWSCLIWith is the internal implementation that accepts an
// injectable runner for testing.
func checkAWSCLIWith(run commandRunner) CheckResult {
	result := CheckResult{Name: "aws-cli"}

	output, err := run("aws", "--version")
	if err != nil {
		result.Status = StatusError
		result.Message = "AWS CLI is not installed or not in PATH"
		return result
	}

	raw := strings.TrimSpace(string(output))
	matches := versionRe.FindStringSubmatch(raw)
	if len(matches) < 2 {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("AWS CLI found but could not parse version from: %s", raw)
		return result
	}

	version := matches[1]
	result.Version = version

	// Check for v2+
	if strings.HasPrefix(version, "1.") {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("AWS CLI v1 detected (%s); v2 is recommended", version)
		return result
	}

	result.Status = StatusOK
	result.Message = fmt.Sprintf("AWS CLI v%s detected", version)
	return result
}
