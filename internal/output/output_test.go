package output

import (
	"strings"
	"testing"
	"time"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
	"github.com/prabhnoor12/dockwhy/internal/docker"
)

func TestJSONUsesStableLowercaseSchema(t *testing.T) {
	var out strings.Builder
	result := diagnosis.Result{Container: docker.Container{ID: "abc", MemoryLimit: 1024}, Reason: "clean exit"}
	if err := JSON(&out, result); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"id": "abc"`) || strings.Contains(out.String(), `"ID"`) {
		t.Fatalf("unexpected JSON schema: %s", out.String())
	}
}

func TestTextMarksTruncatedLogs(t *testing.T) {
	var out strings.Builder
	result := diagnosis.Result{Container: docker.Container{Name: "api"}, Logs: "last line", LogsTruncated: true}
	if err := Text(&out, result); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "[output truncated]") {
		t.Fatalf("missing truncation marker: %s", out.String())
	}
}

func TestTextShowsSecondaryFindings(t *testing.T) {
	var out strings.Builder
	result := diagnosis.Result{
		Container: docker.Container{Name: "api"},
		Reason:    "out of memory",
		Summary:   "memory limit exceeded",
		Findings: []diagnosis.Finding{
			{Rank: 1, Reason: "out of memory", Severity: "critical", Confidence: "high", Summary: "memory limit exceeded"},
			{Rank: 2, Reason: "disk full (secondary signal)", Severity: "warning", Confidence: "low", Summary: "storage may be exhausted", Evidence: []diagnosis.Evidence{{Name: "disk signal", Value: "no space left on device"}}},
		},
	}
	if err := Text(&out, result); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Other findings:") || !strings.Contains(out.String(), "disk full (secondary signal)") {
		t.Fatalf("missing secondary finding: %s", out.String())
	}
}

func TestTextShowsChronologicalLifecycleTimeline(t *testing.T) {
	var out strings.Builder
	started := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC).UnixNano()
	died := time.Date(2026, 8, 29, 10, 0, 5, 0, time.UTC).UnixNano()
	result := diagnosis.Result{
		Container: docker.Container{Name: "api"},
		Events: []docker.Event{
			{TimeNano: died, Action: "die", Attrs: map[string]string{"exitCode": "1"}},
			{TimeNano: started, Action: "start"},
		},
	}
	if err := Text(&out, result); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !strings.Contains(text, "Lifecycle timeline:") || !strings.Contains(text, "exited") {
		t.Fatalf("missing lifecycle timeline: %s", text)
	}
	if strings.Index(text, "started") > strings.Index(text, "exited") {
		t.Fatalf("events were not ordered chronologically: %s", text)
	}
	if !strings.Contains(text, "exitCode=1") {
		t.Fatalf("event attributes were not shown: %s", text)
	}
}

func TestDescribeEvent(t *testing.T) {
	tests := []struct {
		action string
		want   string
	}{
		{"create", "created"},
		{"start", "started"},
		{"stop", "stopped"},
		{"kill", "killed"},
		{"die", "exited"},
		{"restart", "restarted"},
		{"destroy", "destroyed"},
		{"health_status: unhealthy", "health check: unhealthy"},
		{"health_status: healthy", "health check: healthy"},
		{"unknown_action", "unknown_action"},
	}
	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			if got := describeEvent(tt.action); got != tt.want {
				t.Fatalf("describeEvent(%q) = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}

func TestFormatEventTime(t *testing.T) {
	t.Run("zero returns dash", func(t *testing.T) {
		if got := formatEventTime(0); got != "-" {
			t.Fatalf("formatEventTime(0) = %q, want %q", got, "-")
		}
	})
	t.Run("negative returns dash", func(t *testing.T) {
		if got := formatEventTime(-1); got != "-" {
			t.Fatalf("formatEventTime(-1) = %q, want %q", got, "-")
		}
	})
	t.Run("valid timestamp", func(t *testing.T) {
		nano := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC).UnixNano()
		got := formatEventTime(nano)
		if !strings.Contains(got, "2026-08-29") {
			t.Fatalf("formatEventTime = %q, expected 2026-08-29", got)
		}
	})
}

func TestShortID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abcdef0123456789", "abcdef012345"},
		{"short", "short"},
		{"exactly12ch", "exactly12ch"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := shortID(tt.input); got != tt.want {
				t.Fatalf("shortID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatAttrs(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		if got := formatAttrs(nil); got != "" {
			t.Fatalf("formatAttrs(nil) = %q, want empty", got)
		}
	})
	t.Run("sorts keys", func(t *testing.T) {
		attrs := map[string]string{"exitCode": "1", "signal": "15"}
		got := formatAttrs(attrs)
		if got != "exitCode=1 signal=15" {
			t.Fatalf("formatAttrs = %q", got)
		}
	})
}
