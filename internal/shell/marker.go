package shell

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	MarkerStart = "# >>> aws-cli-extension >>>"
	MarkerEnd   = "# <<< aws-cli-extension <<<"
)

// RCFiles returns the standard shell RC file paths to scan.
func RCFiles() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
	}
}

// HasMarker returns true if the file contains the awse marker block.
func HasMarker(rcPath string) bool {
	data, err := os.ReadFile(rcPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), MarkerStart)
}

// RemoveMarker removes the awse marker block from the given RC file.
func RemoveMarker(rcPath string) error {
	data, err := os.ReadFile(rcPath)
	if err != nil {
		return err
	}

	content := string(data)
	startIdx := strings.Index(content, MarkerStart)
	if startIdx == -1 {
		return nil // no marker found
	}

	endIdx := strings.Index(content, MarkerEnd)
	if endIdx == -1 {
		return nil // malformed, skip
	}
	endIdx += len(MarkerEnd)

	// Remove trailing newline if present
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}

	// Also remove leading newline if the block starts after one
	if startIdx > 0 && content[startIdx-1] == '\n' {
		startIdx--
	}

	newContent := content[:startIdx] + content[endIdx:]
	return os.WriteFile(rcPath, []byte(newContent), 0644)
}
