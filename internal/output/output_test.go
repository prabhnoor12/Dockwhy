package output

import (
	"strings"
	"testing"
	"time"

	"github.com/dockwhy/dockwhy/internal/diagnosis"
	"github.com/dockwhy/dockwhy/internal/docker"
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
