package doctor

import (
	"fmt"
	"testing"
)

func fakeRunner(output string, err error) commandRunner {
	return func(name string, args ...string) ([]byte, error) {
		return []byte(output), err
	}
}

func TestCheckAWSCLI_V2OK(t *testing.T) {
	run := fakeRunner("aws-cli/2.15.30 Python/3.11.6 Darwin/23.0.0 source/arm64", nil)
	r := checkAWSCLIWith(run)

	if r.Status != StatusOK {
		t.Errorf("expected StatusOK, got %v", r.Status)
	}
	if r.Version != "2.15.30" {
		t.Errorf("expected version 2.15.30, got %q", r.Version)
	}
	if r.Name != "aws-cli" {
		t.Errorf("expected name aws-cli, got %q", r.Name)
	}
}

func TestCheckAWSCLI_V1Warning(t *testing.T) {
	run := fakeRunner("aws-cli/1.29.10 Python/3.9.0 Linux/5.15.0", nil)
	r := checkAWSCLIWith(run)

	if r.Status != StatusWarning {
		t.Errorf("expected StatusWarning, got %v", r.Status)
	}
	if r.Version != "1.29.10" {
		t.Errorf("expected version 1.29.10, got %q", r.Version)
	}
}

func TestCheckAWSCLI_NotInstalled(t *testing.T) {
	run := fakeRunner("", fmt.Errorf("exec: \"aws\": executable file not found in $PATH"))
	r := checkAWSCLIWith(run)

	if r.Status != StatusError {
		t.Errorf("expected StatusError, got %v", r.Status)
	}
	if r.Version != "" {
		t.Errorf("expected empty version, got %q", r.Version)
	}
}

func TestCheckAWSCLI_UnparseableOutput(t *testing.T) {
	run := fakeRunner("some-unexpected-output", nil)
	r := checkAWSCLIWith(run)

	if r.Status != StatusWarning {
		t.Errorf("expected StatusWarning, got %v", r.Status)
	}
	if r.Version != "" {
		t.Errorf("expected empty version, got %q", r.Version)
	}
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		s    Status
		want string
	}{
		{StatusOK, "ok"},
		{StatusWarning, "warning"},
		{StatusError, "error"},
		{Status(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("Status(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}
