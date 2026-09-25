// Copyright 2026 Prabhnoor12
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package diagnosis

import (
	"errors"
	"strings"
	"testing"

	"github.com/prabhnoor12/dockwhy/internal/docker"
)

func TestAnalyzeOOMKilled(t *testing.T) {
	result := Analyze(docker.Container{State: docker.State{Status: "exited", ExitCode: 137, OOMKilled: true}, MemoryLimit: 128 * 1024 * 1024}, "", nil)
	if result.Reason != "out of memory" {
		t.Fatalf("reason = %q", result.Reason)
	}
	if result.Severity != "critical" {
		t.Fatalf("severity = %q", result.Severity)
	}
}

func TestAnalyzeHealthFailure(t *testing.T) {
	result := Analyze(docker.Container{State: docker.State{Status: "running", Running: true, Health: &docker.Health{Status: "unhealthy", FailingStreak: 3}}}, "", nil)
	if result.Reason != "unhealthy health check" {
		t.Fatalf("reason = %q", result.Reason)
	}
}

func TestAnalyzeCleanExit(t *testing.T) {
	result := Analyze(docker.Container{State: docker.State{Status: "exited", ExitCode: 0}}, "", nil)
	if result.Reason != "clean exit" {
		t.Fatalf("reason = %q", result.Reason)
	}
}

func TestAnalyzeRunningContainer(t *testing.T) {
	result := Analyze(docker.Container{State: docker.State{Status: "running", Running: true}}, "", nil)
	if result.Reason != "container is running" {
		t.Fatalf("reason = %q", result.Reason)
	}
}

func TestAnalyzeOOMRemainsPrimaryWhenDiskSignalExists(t *testing.T) {
	result := Analyze(docker.Container{State: docker.State{Status: "exited", ExitCode: 137, OOMKilled: true}}, "no space left on device", nil)
	if result.Reason != "out of memory" {
		t.Fatalf("reason = %q", result.Reason)
	}
	if len(result.Evidence) < 1 {
		t.Fatal("expected secondary disk evidence")
	}
	if len(result.Findings) < 2 || result.Findings[0].Reason != "out of memory" || result.Findings[1].Reason != "disk full (secondary signal)" {
		t.Fatalf("unexpected ranked findings: %#v", result.Findings)
	}
}

func TestAnalyzeDiskFullFromLogs(t *testing.T) {
	result := Analyze(docker.Container{State: docker.State{Status: "exited", ExitCode: 1}}, "write failed: no space left on device", nil)
	if result.Reason != "disk full" {
		t.Fatalf("reason = %q", result.Reason)
	}
}

func TestAnalyzeDetailedIncludesEventsChronologically(t *testing.T) {
	events := []docker.Event{
		{TimeNano: 200, Action: "die"},
		{TimeNano: 100, Action: "start"},
	}
	result := AnalyzeDetailed(
		docker.Container{State: docker.State{Status: "exited", ExitCode: 0}},
		"", false, events, nil, nil,
	)
	if len(result.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result.Events))
	}
	if result.Events[0].Action != "start" || result.Events[1].Action != "die" {
		t.Fatalf("events not sorted chronologically: %#v", result.Events)
	}
}

func TestAnalyzeDetailedCapturesLogTruncation(t *testing.T) {
	result := AnalyzeDetailed(
		docker.Container{State: docker.State{Status: "exited", ExitCode: 0}},
		"some logs", true, nil, nil, nil,
	)
	if !result.LogsTruncated {
		t.Fatal("expected LogsTruncated to be true")
	}
}

func TestAnalyzeDetailedCapturesErrors(t *testing.T) {
	logErr := errors.New("log read failed")
	eventsErr := errors.New("events unavailable")
	result := AnalyzeDetailed(
		docker.Container{State: docker.State{Status: "exited", ExitCode: 0}},
		"", false, nil, logErr, eventsErr,
	)
	if result.LogError != "log read failed" {
		t.Fatalf("LogError = %q", result.LogError)
	}
	if result.EventsError != "events unavailable" {
		t.Fatalf("EventsError = %q", result.EventsError)
	}
}

func TestAnalyzeDetailedComposeAdvice(t *testing.T) {
	result := AnalyzeDetailed(
		docker.Container{
			State:          docker.State{Status: "exited", ExitCode: 1},
			ComposeService: "api",
			ComposeProject: "shop",
		}, "", false, nil, nil, nil,
	)
	found := false
	for _, advice := range result.Advice {
		if strings.Contains(advice, "docker compose logs api") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected compose advice, got: %v", result.Advice)
	}
}

func TestAnalyzeDetailedRestartCountAdvice(t *testing.T) {
	result := AnalyzeDetailed(
		docker.Container{State: docker.State{Status: "exited", ExitCode: 1}, RestartCount: 5},
		"", false, nil, nil, nil,
	)
	found := false
	for _, advice := range result.Advice {
		if strings.Contains(advice, "restarted 5 time(s)") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected restart advice, got: %v", result.Advice)
	}
}

func TestAnalyzeDetailedExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		reason   string
	}{
		{"exit 126", 126, "command not executable (exit 126)"},
		{"exit 127", 127, "command not found (exit 127)"},
		{"exit 139", 139, "segmentation fault (exit 139)"},
		{"exit 143", 143, "gracefully stopped (exit 143)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Analyze(docker.Container{State: docker.State{Status: "exited", ExitCode: tt.exitCode}}, "", nil)
			if result.Reason != tt.reason {
				t.Fatalf("exit %d: reason = %q, want %q", tt.exitCode, result.Reason, tt.reason)
			}
		})
	}
}

func TestAnalyzeDetailedFindingsAreRanked(t *testing.T) {
	result := AnalyzeDetailed(
		docker.Container{State: docker.State{Status: "exited", ExitCode: 137, OOMKilled: true}},
		"no space left on device", false, nil, nil, nil,
	)
	if len(result.Findings) < 2 {
		t.Fatalf("expected at least 2 findings, got %d", len(result.Findings))
	}
	for i, f := range result.Findings {
		if f.Rank != i+1 {
			t.Fatalf("finding %d has rank %d", i, f.Rank)
		}
	}
	if result.Findings[0].Reason != "out of memory" {
		t.Fatalf("primary finding should be OOM, got %q", result.Findings[0].Reason)
	}
}
