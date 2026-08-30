package docker

import (
	"strings"
	"testing"
)

func TestInspectContainerNormalizesDockerData(t *testing.T) {
	raw := inspectContainer{
		ID: "abcdef0123456789", Name: "/my-api", Created: "2026-08-29T10:00:00Z", RestartCount: 4,
		Config: inspectConfig{Image: "api:latest", Labels: map[string]string{"team": "platform"},
			Entrypoint: []string{"/bin/sh", "-c"}, Cmd: []string{"./start.sh"}, WorkingDir: "/app", User: "1000", StopSignal: "SIGTERM"},
		State: inspectState{Status: "exited", ExitCode: 137, OOMKilled: true,
			StartedAt: "2026-08-29T10:01:00Z", FinishedAt: "2026-08-29T10:02:00Z",
			Health: &inspectHealth{Status: "unhealthy", FailingStreak: 2, Log: []HealthCheck{{ExitCode: 1, Output: "timeout"}}}},
		Host: inspectHostConfig{Memory: 128 * 1024 * 1024, MemorySwap: -1, ReadonlyRootfs: true,
			LogConfig: struct {
				Type string `json:"Type"`
			}{Type: "json-file"},
			StorageOpt: map[string]string{"size": "10G"}},
		Mounts: []inspectMount{{Type: "bind", Source: "/host/config", Destination: "/app/config", RW: false}},
		SizeRW: 4096, SizeRoot: 256 * 1024 * 1024,
	}

	container := raw.container()
	if container.Name != "my-api" || container.RestartCount != 4 || !container.State.OOMKilled {
		t.Fatalf("unexpected identity/state: %#v", container)
	}
	if container.DiskLimit != "10G" || container.MemoryLimit != 128*1024*1024 || !container.DiskReadOnly {
		t.Fatalf("unexpected limits: %#v", container)
	}
	if container.LogDriver != "json-file" || container.WorkingDir != "/app" || container.User != "1000" || len(container.Mounts) != 1 {
		t.Fatalf("unexpected runtime configuration: %#v", container)
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

func TestStatsRecordNormalizesDockerStats(t *testing.T) {
	stats, err := (statsRecord{
		CPUPercent: "12.50%", MemUsage: "128MiB / 1GiB", MemPercent: "12.50%",
		NetIO: "1.5MB / 2MiB", BlockIO: "3kB / 4B", PIDs: "7",
	}).stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.CPUPercent != 12.5 || stats.MemoryUsageBytes != 128*1024*1024 || stats.MemoryLimitBytes != 1024*1024*1024 {
		t.Fatalf("unexpected CPU/memory stats: %#v", stats)
	}
	if stats.NetworkRxBytes != 1500000 || stats.NetworkTxBytes != 2*1024*1024 || stats.PidsCurrent != 7 {
		t.Fatalf("unexpected I/O/process stats: %#v", stats)
	}
}
