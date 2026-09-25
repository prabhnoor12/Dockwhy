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

package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prabhnoor12/dockwhy/internal/docker"
	"github.com/prabhnoor12/dockwhy/internal/kube"
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
func (f fakeClient) ListByProject(string) ([]docker.ContainerSummary, error) {
	return nil, nil
}

func TestRunHumanOutput(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run(context.Background(),[]string{"--tail", "10", "my-api"}, &stdout, &stderr, fakeClient{
		container: docker.Container{Name: "my-api", ID: "0123456789abcdef", Image: "api:latest", State: docker.State{Status: "exited", ExitCode: 1}},
		logs:      "fatal: configuration missing",
	}, nil)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "application exited") || !strings.Contains(stdout.String(), "configuration missing") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunInspectError(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := run(context.Background(),[]string{"my-api"}, &stdout, &stderr, failingClient{}, nil); code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunNoLogsSkipsLogCollection(t *testing.T) {
	var stdout, stderr strings.Builder
	logCalls := 0
	code := run(context.Background(),[]string{"--no-logs", "my-api"}, &stdout, &stderr, fakeClient{
		container: docker.Container{Name: "my-api", ID: "container-id", Image: "api:latest", State: docker.State{Status: "exited", ExitCode: 1}},
		logCalls:  &logCalls,
	}, nil)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if logCalls != 0 || !strings.Contains(stdout.String(), "skipped by --no-logs") {
		t.Fatalf("logs were not skipped: calls=%d output=%s", logCalls, stdout.String())
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := run(context.Background(),[]string{"--version"}, &stdout, &stderr, fakeClient{}, nil); code != 0 {
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
func (failingClient) ListByProject(string) ([]docker.ContainerSummary, error) {
	return nil, errors.New("project not found")
}

func TestRunTailNegative(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := run(context.Background(),[]string{"--tail", "-1", "my-api"}, &stdout, &stderr, fakeClient{}, nil); code != 2 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "-tail must be zero or greater") {
		t.Fatalf("expected tail validation error: %q", stderr.String())
	}
}

func TestRunTailExceedsMaximum(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := run(context.Background(),[]string{"--tail", "5001", "my-api"}, &stdout, &stderr, fakeClient{}, nil); code != 2 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "-tail must not exceed 5000") {
		t.Fatalf("expected tail maximum error: %q", stderr.String())
	}
}

func TestRunTimeoutZero(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := run(context.Background(),[]string{"--timeout", "0s", "my-api"}, &stdout, &stderr, fakeClient{}, nil); code != 2 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "-timeout and -events-since must be greater than zero") {
		t.Fatalf("expected timeout validation error: %q", stderr.String())
	}
}

func TestRunEventsSinceZero(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := run(context.Background(),[]string{"--events-since", "0s", "my-api"}, &stdout, &stderr, fakeClient{}, nil); code != 2 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "-timeout and -events-since must be greater than zero") {
		t.Fatalf("expected events-since validation error: %q", stderr.String())
	}
}

func TestRunMissingContainer(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := run(context.Background(),[]string{}, &stdout, &stderr, fakeClient{}, nil); code != 2 {
		t.Fatalf("code = %d", code)
	}
}

func TestRunJSONOutput(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run(context.Background(),[]string{"--json", "my-api"}, &stdout, &stderr, fakeClient{
		container: docker.Container{Name: "my-api", ID: "abc123", Image: "api:latest", State: docker.State{Status: "exited", ExitCode: 0}},
	}, nil)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"reason"`) || !strings.Contains(stdout.String(), `"container"`) {
		t.Fatalf("expected JSON output: %s", stdout.String())
	}
}

func TestLookupFlagValue(t *testing.T) {
	tests := []struct {
		args  []string
		name  string
		want  string
	}{
		{[]string{"--timeout", "30s", "my-api"}, "timeout", "30s"},
		{[]string{"-timeout", "30s", "my-api"}, "timeout", "30s"},
		{[]string{"--timeout=30s", "my-api"}, "timeout", "30s"},
		{[]string{"-timeout=30s", "my-api"}, "timeout", "30s"},
		{[]string{"my-api"}, "timeout", ""},
		{[]string{"--tail", "100", "my-api"}, "timeout", ""},
		{[]string{"--timeout"}, "timeout", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name+"="+tt.want, func(t *testing.T) {
			got := lookupFlagValue(tt.args, tt.name)
			if got != tt.want {
				t.Fatalf("lookupFlagValue(%v, %q) = %q, want %q", tt.args, tt.name, got, tt.want)
			}
		})
	}
}

type projectClient struct {
	fakeClient
	summaries []docker.ContainerSummary
}

func (p projectClient) ListByProject(string) ([]docker.ContainerSummary, error) {
	return p.summaries, nil
}

func TestRunProjectDiagnosesAllContainers(t *testing.T) {
	var stdout, stderr strings.Builder
	client := projectClient{
		fakeClient: fakeClient{
			container: docker.Container{Name: "shop-api-1", ID: "abc", Image: "api:latest", ComposeService: "api", State: docker.State{Status: "exited", ExitCode: 1}},
		},
		summaries: []docker.ContainerSummary{
			{Name: "shop-api-1", ComposeService: "api"},
		},
	}
	code := run(context.Background(),[]string{"--project", "shop"}, &stdout, &stderr, client, nil)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "shop") || !strings.Contains(stdout.String(), "Containers:") {
		t.Fatalf("expected project output: %s", stdout.String())
	}
}

func TestRunProjectNotFound(t *testing.T) {
	var stdout, stderr strings.Builder
	client := projectClient{
		fakeClient: fakeClient{},
		summaries:  nil,
	}
	code := run(context.Background(),[]string{"--project", "missing"}, &stdout, &stderr, client, nil)
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr.String(), "no containers found") {
		t.Fatalf("expected error: %q", stderr.String())
	}
}

type fakeKubeClient struct {
	pod       kube.PodInfo
	container string
}

func (f fakeKubeClient) GetPod(namespace, name string) (kube.PodInfo, error) {
	return f.pod, nil
}
func (f fakeKubeClient) ResolveContainer(podName, containerName, namespace string) (string, error) {
	return f.container, nil
}
func (f fakeKubeClient) DetectNamespace() string {
	return "default"
}

func TestRunKubeMode(t *testing.T) {
	var stdout, stderr strings.Builder
	kubeClient := fakeKubeClient{
		pod: kube.PodInfo{
			Name:      "api-pod",
			Namespace: "production",
			Node:      "node-1",
			Phase:     "Running",
		},
		container: "resolved-container-id",
	}
	code := run(context.Background(),[]string{"--kube", "api-pod"}, &stdout, &stderr, fakeClient{
		container: docker.Container{Name: "resolved-container-id", ID: "resolved-container-id", Image: "api:latest", State: docker.State{Status: "exited", ExitCode: 1}},
		logs:      "error: connection refused",
	}, kubeClient)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Kubernetes Pod: production/api-pod") {
		t.Fatalf("expected kube context in output: %s", out)
	}
}

func TestRunKubeModeJSON(t *testing.T) {
	var stdout, stderr strings.Builder
	kubeClient := fakeKubeClient{
		pod: kube.PodInfo{
			Name:      "api-pod",
			Namespace: "production",
			Phase:     "Running",
		},
		container: "resolved-id",
	}
	code := run(context.Background(),[]string{"--kube", "--json", "api-pod"}, &stdout, &stderr, fakeClient{
		container: docker.Container{Name: "resolved-id", ID: "resolved-id", Image: "api:latest", State: docker.State{Status: "exited", ExitCode: 0}},
	}, kubeClient)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"kube_pod"`) {
		t.Fatalf("expected kube_pod in JSON: %s", out)
	}
	if !strings.Contains(out, `"api-pod"`) {
		t.Fatalf("expected pod name in JSON: %s", out)
	}
}

func TestRunWatchMode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var stdout, stderr strings.Builder
	go func() {
		for i := 0; i < 50; i++ {
			time.Sleep(50 * time.Millisecond)
			if strings.Count(stdout.String(), "===") >= 2 {
				cancel()
				return
			}
		}
		cancel()
	}()
	code := run(ctx, []string{"--watch", "50ms", "my-api"}, &stdout, &stderr, fakeClient{
		container: docker.Container{Name: "my-api", ID: "abc", Image: "api:latest", State: docker.State{Status: "exited", ExitCode: 1}},
		logs:      "error starting",
	}, nil)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "===") {
		t.Fatalf("expected watch separator in output: %s", out)
	}
	count := strings.Count(out, "===")
	if count < 2 {
		t.Fatalf("expected at least 2 watch iterations, got %d", count)
	}
}
