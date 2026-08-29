package diagnosis

import (
	"testing"

	"github.com/dockwhy/dockwhy/internal/docker"
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
}

func TestAnalyzeDiskFullFromLogs(t *testing.T) {
	result := Analyze(docker.Container{State: docker.State{Status: "exited", ExitCode: 1}}, "write failed: no space left on device", nil)
	if result.Reason != "disk full" {
		t.Fatalf("reason = %q", result.Reason)
	}
}
