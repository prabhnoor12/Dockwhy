package docker

import (
	"strings"
	"testing"
)

func TestInspectContainerNormalizesDockerData(t *testing.T) {
	raw := inspectContainer{
		ID: "abcdef0123456789", Name: "/my-api", Created: "2026-08-29T10:00:00Z", RestartCount: 4,
		Config: inspectConfig{Image: "api:latest", Labels: map[string]string{"team": "platform"}},
		State: inspectState{Status: "exited", ExitCode: 137, OOMKilled: true,
			StartedAt: "2026-08-29T10:01:00Z", FinishedAt: "2026-08-29T10:02:00Z",
			Health: &inspectHealth{Status: "unhealthy", FailingStreak: 2, Log: []HealthCheck{{ExitCode: 1, Output: "timeout"}}}},
		Host: inspectHostConfig{Memory: 128 * 1024 * 1024, MemorySwap: -1, ReadonlyRootfs: true,
			StorageOpt: map[string]string{"size": "10G"}},
		SizeRW: 4096, SizeRoot: 256 * 1024 * 1024,
	}

	container := raw.container()
	if container.Name != "my-api" || container.RestartCount != 4 || !container.State.OOMKilled {
		t.Fatalf("unexpected identity/state: %#v", container)
	}
	if container.DiskLimit != "10G" || container.MemoryLimit != 128*1024*1024 || !container.DiskReadOnly {
		t.Fatalf("unexpected limits: %#v", container)
	}
	if container.State.Health == nil || container.State.Health.Log[0].Output != "timeout" {
		t.Fatalf("health check was not normalized: %#v", container.State.Health)
	}
}

func TestLimitedBufferBoundsOutput(t *testing.T) {
	var buffer limitedBuffer
	buffer.limit = 4
	if _, err := buffer.Write([]byte("0123456789")); err != nil {
		t.Fatal(err)
	}
	if buffer.String() != "0123" || !buffer.truncated {
		t.Fatalf("buffer = %q, truncated = %v", buffer.String(), buffer.truncated)
	}
	if _, err := buffer.Write([]byte("more")); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(buffer.String(), "0123") {
		t.Fatalf("buffer grew past limit: %q", buffer.String())
	}
}
