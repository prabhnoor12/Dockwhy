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
}

func (f fakeClient) Inspect(string) (docker.Container, error) { return f.container, nil }
func (f fakeClient) Logs(string, int) (docker.LogOutput, error) {
	return docker.LogOutput{Text: f.logs}, nil
}
func (f fakeClient) Events(string, time.Duration) ([]docker.Event, error) { return nil, nil }

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

type failingClient struct{}

func (failingClient) Inspect(string) (docker.Container, error) {
	return docker.Container{}, errors.New("container not found")
}
func (failingClient) Logs(string, int) (docker.LogOutput, error)           { return docker.LogOutput{}, nil }
func (failingClient) Events(string, time.Duration) ([]docker.Event, error) { return nil, nil }
