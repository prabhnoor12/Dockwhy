package diagnosis

import (
	"testing"
)

func TestLookupExitCodeKnown(t *testing.T) {
	tests := []struct {
		code     int
		name     string
		category string
	}{
		{0, "success", "success"},
		{1, "general_error", "application"},
		{126, "command_not_executable", "configuration"},
		{127, "command_not_found", "configuration"},
		{137, "sigkill", "signal"},
		{139, "sigsegv", "signal"},
		{143, "sigterm", "signal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := LookupExitCode(tt.code)
			if info.Code != tt.code {
				t.Errorf("code = %d, want %d", info.Code, tt.code)
			}
			if info.Name != tt.name {
				t.Errorf("name = %q, want %q", info.Name, tt.name)
			}
			if info.Category != tt.category {
				t.Errorf("category = %q, want %q", info.Category, tt.category)
			}
			if info.Description == "" {
				t.Error("description should not be empty")
			}
			if len(info.Causes) == 0 {
				t.Error("causes should not be empty")
			}
			if len(info.Fixes) == 0 {
				t.Error("fixes should not be empty")
			}
		})
	}
}

func TestLookupExitCodeUnknown(t *testing.T) {
	info := LookupExitCode(42)
	if info.Name != "unknown" {
		t.Errorf("name = %q, want %q", info.Name, "unknown")
	}
	if info.Category != "application" {
		t.Errorf("category = %q, want %q", info.Category, "application")
	}
}

func TestLookupExitCodeSignalFallback(t *testing.T) {
	info := LookupExitCode(130)
	if info.Signal != "SIGINT" {
		t.Errorf("signal = %q, want %q", info.Signal, "SIGINT")
	}
	if info.Category != "signal" {
		t.Errorf("category = %q, want %q", info.Category, "signal")
	}
}

func TestLookupExitCode137Details(t *testing.T) {
	info := LookupExitCode(137)
	if info.Signal != "SIGKILL" {
		t.Errorf("signal = %q, want SIGKILL", info.Signal)
	}
	found := false
	for _, cause := range info.Causes {
		if cause == "OOM killer terminated the process (most common)" {
			found = true
		}
	}
	if !found {
		t.Error("137 should mention OOM killer as a cause")
	}
}

func TestAllExitCodes(t *testing.T) {
	codes := AllExitCodes()
	if len(codes) < 10 {
		t.Errorf("expected at least 10 exit codes, got %d", len(codes))
	}
	for _, info := range codes {
		if info.Description == "" {
			t.Errorf("exit code %d has empty description", info.Code)
		}
	}
}
