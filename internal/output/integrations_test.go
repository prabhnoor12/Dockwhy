package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
	"github.com/prabhnoor12/dockwhy/internal/docker"
)

func testResult() diagnosis.Result {
	return diagnosis.Result{
		Container: docker.Container{
			ID:             "abc123",
			Name:           "my-api",
			Image:          "api:latest",
			ComposeProject: "backend",
			ComposeService: "api",
			State: docker.State{
				Status:    "exited",
				ExitCode:  137,
				OOMKilled: true,
			},
			RestartCount: 3,
		},
		Reason:     "OOM killed",
		Summary:    "Container was killed by the OOM killer",
		Severity:   "critical",
		Confidence: "high",
		Advice:     []string{"Increase memory limit", "Check for memory leaks"},
		Stats: &docker.Stats{
			CPUPercent:       45.2,
			MemoryUsageBytes: 536870912,
		},
	}
}

func TestPagerDuty(t *testing.T) {
	var buf bytes.Buffer
	if err := PagerDuty(&buf, testResult()); err != nil {
		t.Fatal(err)
	}
	var payload PagerDutyPayload
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if payload.EventAction != "trigger" {
		t.Errorf("event_action = %q, want %q", payload.EventAction, "trigger")
	}
	if payload.Payload.Severity != "critical" {
		t.Errorf("severity = %q, want %q", payload.Payload.Severity, "critical")
	}
	if payload.Payload.Source != "my-api" {
		t.Errorf("source = %q, want %q", payload.Payload.Source, "my-api")
	}
	if payload.Payload.Component != "api" {
		t.Errorf("component = %q, want %q", payload.Payload.Component, "api")
	}
	if payload.Payload.Group != "backend" {
		t.Errorf("group = %q, want %q", payload.Payload.Group, "backend")
	}
	if !strings.Contains(payload.Payload.Summary, "my-api") {
		t.Errorf("summary should contain container name: %q", payload.Payload.Summary)
	}
}

func TestPagerDutySeverityMapping(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"critical", "critical"},
		{"error", "error"},
		{"warning", "warning"},
		{"info", "info"},
		{"unknown", "info"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := testResult()
			r.Severity = tt.input
			var buf bytes.Buffer
			if err := PagerDuty(&buf, r); err != nil {
				t.Fatal(err)
			}
			var payload PagerDutyPayload
			if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Payload.Severity != tt.want {
				t.Errorf("severity = %q, want %q", payload.Payload.Severity, tt.want)
			}
		})
	}
}

func TestSlack(t *testing.T) {
	var buf bytes.Buffer
	if err := Slack(&buf, testResult()); err != nil {
		t.Fatal(err)
	}
	var payload SlackPayload
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !strings.Contains(payload.Text, "my-api") {
		t.Errorf("text should contain container name: %q", payload.Text)
	}
	if len(payload.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(payload.Attachments))
	}
	att := payload.Attachments[0]
	if att.Color != "#e01e5a" {
		t.Errorf("color = %q, want %q (critical)", att.Color, "#e01e5a")
	}
	if len(att.Fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(att.Fields))
	}
}

func TestSlackColorMapping(t *testing.T) {
	tests := []struct {
		severity string
		color    string
	}{
		{"critical", "#e01e5a"},
		{"error", "#e89216"},
		{"warning", "#f2c744"},
		{"info", "#36a64f"},
	}
	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			r := testResult()
			r.Severity = tt.severity
			var buf bytes.Buffer
			if err := Slack(&buf, r); err != nil {
				t.Fatal(err)
			}
			var payload SlackPayload
			if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Attachments[0].Color != tt.color {
				t.Errorf("color = %q, want %q", payload.Attachments[0].Color, tt.color)
			}
		})
	}
}

func TestPrometheus(t *testing.T) {
	var buf bytes.Buffer
	if err := Prometheus(&buf, testResult()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	expected := []string{
		"# HELP dockwhy_exit_code",
		"# TYPE dockwhy_exit_code gauge",
		`dockwhy_exit_code{container="my-api"`,
		"137",
		"# HELP dockwhy_oom_killed",
		`dockwhy_oom_killed{container="my-api"} 1`,
		"# HELP dockwhy_restart_count",
		`dockwhy_restart_count{container="my-api"} 3`,
		"# HELP dockwhy_severity",
		`dockwhy_severity{container="my-api"`,
		"3",
		"# HELP dockwhy_cpu_percent",
		`dockwhy_cpu_percent{container="my-api"} 45.20`,
		"# HELP dockwhy_memory_usage_bytes",
		`dockwhy_memory_usage_bytes{container="my-api"} 536870912`,
	}
	for _, s := range expected {
		if !strings.Contains(out, s) {
			t.Errorf("missing %q in output:\n%s", s, out)
		}
	}
}

func TestPrometheusWithoutStats(t *testing.T) {
	r := testResult()
	r.Stats = nil
	var buf bytes.Buffer
	if err := Prometheus(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "dockwhy_cpu_percent") {
		t.Error("should not contain cpu metric without stats")
	}
	if strings.Contains(out, "dockwhy_memory_usage_bytes") {
		t.Error("should not contain memory metric without stats")
	}
}

func TestPrometheusOOMFalse(t *testing.T) {
	r := testResult()
	r.Container.State.OOMKilled = false
	var buf bytes.Buffer
	if err := Prometheus(&buf, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `dockwhy_oom_killed{container="my-api"} 0`) {
		t.Error("OOM killed should be 0 when not OOM killed")
	}
}

func TestWebhook(t *testing.T) {
	var buf bytes.Buffer
	if err := Webhook(&buf, testResult()); err != nil {
		t.Fatal(err)
	}
	var payload WebhookPayload
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if payload.Source != "dockwhy" {
		t.Errorf("source = %q, want %q", payload.Source, "dockwhy")
	}
	if payload.Severity != "critical" {
		t.Errorf("severity = %q, want %q", payload.Severity, "critical")
	}
	if payload.Reason != "OOM killed" {
		t.Errorf("reason = %q, want %q", payload.Reason, "OOM killed")
	}
	if payload.Container.Name != "my-api" {
		t.Errorf("container name = %q, want %q", payload.Container.Name, "my-api")
	}
	if payload.Container.ExitCode != 137 {
		t.Errorf("exit code = %d, want %d", payload.Container.ExitCode, 137)
	}
	if !payload.Container.OOMKilled {
		t.Error("oom_killed should be true")
	}
	if payload.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
	if len(payload.Advice) != 2 {
		t.Errorf("expected 2 advice items, got %d", len(payload.Advice))
	}
}

func TestSeverityInt(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"critical", 3},
		{"error", 2},
		{"warning", 1},
		{"info", 0},
		{"", 0},
		{"unknown", 0},
	}
	for _, tt := range tests {
		if got := severityInt(tt.input); got != tt.want {
			t.Errorf("severityInt(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
