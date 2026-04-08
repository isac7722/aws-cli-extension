// Package doctor provides health-check utilities.
// This file implements OS detection and platform-specific AWS CLI
// installation guidance.
package doctor

import "runtime"

// Platform represents a detected operating system.
type Platform int

const (
	PlatformMacOS Platform = iota
	PlatformLinux
	PlatformWindows
	PlatformUnknown
)

// String returns a human-readable name for the platform.
func (p Platform) String() string {
	switch p {
	case PlatformMacOS:
		return "macOS"
	case PlatformLinux:
		return "Linux"
	case PlatformWindows:
		return "Windows"
	default:
		return "Unknown"
	}
}

// DetectPlatform returns the current operating system as a Platform value.
func DetectPlatform() Platform {
	return detectPlatformFrom(runtime.GOOS)
}

// detectPlatformFrom maps a GOOS string to a Platform (testable).
func detectPlatformFrom(goos string) Platform {
	switch goos {
	case "darwin":
		return PlatformMacOS
	case "linux":
		return PlatformLinux
	case "windows":
		return PlatformWindows
	default:
		return PlatformUnknown
	}
}

// InstallGuidance holds platform-specific AWS CLI v2 installation instructions.
type InstallGuidance struct {
	Platform Platform
	Title    string   // short heading, e.g. "Install AWS CLI v2 on macOS"
	Steps    []string // ordered install steps
	URL      string   // official documentation link
}

// GetInstallGuidance returns installation instructions appropriate for
// the current OS.
func GetInstallGuidance() InstallGuidance {
	return GetInstallGuidanceFor(DetectPlatform())
}

// GetInstallGuidanceFor returns installation instructions for the given platform.
func GetInstallGuidanceFor(p Platform) InstallGuidance {
	switch p {
	case PlatformMacOS:
		return InstallGuidance{
			Platform: PlatformMacOS,
			Title:    "Install AWS CLI v2 on macOS",
			Steps: []string{
				"brew install awscli",
				"  — or download the official .pkg installer:",
				"  curl \"https://awscli.amazonaws.com/AWSCLIV2.pkg\" -o \"AWSCLIV2.pkg\"",
				"  sudo installer -pkg AWSCLIV2.pkg -target /",
			},
			URL: "https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2-mac.html",
		}
	case PlatformLinux:
		return InstallGuidance{
			Platform: PlatformLinux,
			Title:    "Install AWS CLI v2 on Linux",
			Steps: []string{
				"curl \"https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip\" -o \"awscliv2.zip\"",
				"unzip awscliv2.zip",
				"sudo ./aws/install",
			},
			URL: "https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2-linux.html",
		}
	case PlatformWindows:
		return InstallGuidance{
			Platform: PlatformWindows,
			Title:    "Install AWS CLI v2 on Windows",
			Steps: []string{
				"Download the MSI installer from:",
				"  https://awscli.amazonaws.com/AWSCLIV2.msi",
				"Run the installer and follow the prompts.",
			},
			URL: "https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2-windows.html",
		}
	default:
		return InstallGuidance{
			Platform: PlatformUnknown,
			Title:    "Install AWS CLI v2",
			Steps: []string{
				"Visit the official installation guide for your platform.",
			},
			URL: "https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html",
		}
	}
}
