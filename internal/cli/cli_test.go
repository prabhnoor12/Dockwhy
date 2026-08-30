package cli

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dockwhy/dockwhy/internal/docker"
)

type fakeClient struct {
	container docker.Container
	logs      string
	logCalls  *int
}

func (f fakeClient) Inspect(string) (docker.Container, error) { return f.container, nil }
func (f fakeClient) Logs(string, int) (docker.LogOutput, error) {
	if f.logCalls != nil {
		*f.logCalls++
	}
	return docker.LogOutput{Text: f.logs}, nil
}
func (f fakeClient) Events(string, time.Duration) ([]docker.Event, error) { return nil, nil }
func (f fakeClient) Stats(string) (docker.Stats, error)                   { return docker.Stats{}, nil }

func TestRunHumanOutput(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run([]string{"--tail", "10", "my-api"}, &stdout, &stderr, fakeClient{
		container: docker.Container{Name: "my-api", ID: "0123456789abcdef", Image: "api:latest", State: docker.State{Status: "exited", ExitCode: 1}},
		logs:      "fatal: configuration missing",
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "application exited") || !strings.Contains(stdout.String(), "configuration missing") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunInspectError(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := run([]string{"my-api"}, &stdout, &stderr, failingClient{}); code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunNoLogsSkipsLogCollection(t *testing.T) {
	var stdout, stderr strings.Builder
	logCalls := 0
	code := run([]string{"--no-logs", "my-api"}, &stdout, &stderr, fakeClient{
		container: docker.Container{Name: "my-api", ID: "container-id", Image: "api:latest", State: docker.State{Status: "exited", ExitCode: 1}},
		logCalls:  &logCalls,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if logCalls != 0 || !strings.Contains(stdout.String(), "skipped by --no-logs") {
		t.Fatalf("logs were not skipped: calls=%d output=%s", logCalls, stdout.String())
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := run([]string{"--version"}, &stdout, &stderr, fakeClient{}); code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) == "" {
		t.Fatal("version output is empty")
	}
}

type failingClient struct{}

func (failingClient) Inspect(string) (docker.Container, error) {
	return docker.Container{}, errors.New("container not found")
}
func (failingClient) Logs(string, int) (docker.LogOutput, error)           { return docker.LogOutput{}, nil }
func (failingClient) Events(string, time.Duration) ([]docker.Event, error) { return nil, nil }
func (failingClient) Stats(string) (docker.Stats, error)                   { return docker.Stats{}, nil }
