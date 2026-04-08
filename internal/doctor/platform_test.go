package doctor

import "testing"

func TestDetectPlatformFrom(t *testing.T) {
	tests := []struct {
		goos string
		want Platform
	}{
		{"darwin", PlatformMacOS},
		{"linux", PlatformLinux},
		{"windows", PlatformWindows},
		{"freebsd", PlatformUnknown},
		{"", PlatformUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			got := detectPlatformFrom(tt.goos)
			if got != tt.want {
				t.Errorf("detectPlatformFrom(%q) = %v, want %v", tt.goos, got, tt.want)
			}
		})
	}
}

func TestPlatformString(t *testing.T) {
	tests := []struct {
		p    Platform
		want string
	}{
		{PlatformMacOS, "macOS"},
		{PlatformLinux, "Linux"},
		{PlatformWindows, "Windows"},
		{PlatformUnknown, "Unknown"},
		{Platform(99), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.p.String(); got != tt.want {
				t.Errorf("Platform(%d).String() = %q, want %q", tt.p, got, tt.want)
			}
		})
	}
}

func TestGetInstallGuidanceFor(t *testing.T) {
	platforms := []Platform{PlatformMacOS, PlatformLinux, PlatformWindows, PlatformUnknown}

	for _, p := range platforms {
		t.Run(p.String(), func(t *testing.T) {
			g := GetInstallGuidanceFor(p)

			if g.Platform != p {
				t.Errorf("guidance.Platform = %v, want %v", g.Platform, p)
			}
			if g.Title == "" {
				t.Error("guidance.Title is empty")
			}
			if len(g.Steps) == 0 {
				t.Error("guidance.Steps is empty")
			}
			if g.URL == "" {
				t.Error("guidance.URL is empty")
			}
		})
	}
}

func TestGetInstallGuidance(t *testing.T) {
	// Just verify it doesn't panic and returns something valid.
	g := GetInstallGuidance()
	if g.Title == "" {
		t.Error("GetInstallGuidance() returned empty title")
	}
}
