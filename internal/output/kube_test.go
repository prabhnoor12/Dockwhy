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

package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
	"github.com/prabhnoor12/dockwhy/internal/docker"
	"github.com/prabhnoor12/dockwhy/internal/kube"
)

func TestKubeContext(t *testing.T) {
	pod := kube.PodInfo{
		Name:      "api-pod-abc",
		Namespace: "production",
		Node:      "node-1",
		Phase:     "Running",
		QoS:       "Burstable",
		Labels:    map[string]string{"app": "api"},
		Restarts:  5,
		Containers: []kube.PodContainer{
			{
				Name:         "api",
				Image:        "api:v2",
				State:        "running",
				Ready:        true,
				RestartCount: 3,
				LastReason:   "OOMKilled",
				LastExitCode: 137,
				LastOOM:      true,
			},
			{
				Name:  "sidecar",
				Image: "envoy:latest",
				State: "waiting",
				Ready: false,
			},
		},
	}
	var buf bytes.Buffer
	KubeContext(&buf, pod)
	out := buf.String()

	expected := []string{
		"Kubernetes Pod: production/api-pod-abc",
		"Node:   node-1",
		"Phase:  Running",
		"QoS:    Burstable",
		"Total Restarts: 5",
		"Containers:",
		"api [api:v2] running, 3 restart(s), last: OOMKilled (exit 137) [OOM]",
		"sidecar [envoy:latest] waiting (not ready)",
	}
	for _, s := range expected {
		if !strings.Contains(out, s) {
			t.Errorf("missing %q in output:\n%s", s, out)
		}
	}
}

func TestKubeContextMinimal(t *testing.T) {
	pod := kube.PodInfo{
		Name:      "simple-pod",
		Namespace: "default",
		Phase:     "Failed",
	}
	var buf bytes.Buffer
	KubeContext(&buf, pod)
	out := buf.String()
	if !strings.Contains(out, "default/simple-pod") {
		t.Errorf("missing pod name: %s", out)
	}
	if strings.Contains(out, "Node:") {
		t.Error("should not show node when empty")
	}
	if strings.Contains(out, "Containers:") {
		t.Error("should not show containers when empty")
	}
}

func TestJSONWithKube(t *testing.T) {
	result := diagnosis.Result{
		Container: docker.Container{Name: "test", ID: "abc", Image: "img", State: docker.State{Status: "exited", ExitCode: 1}},
		Reason:    "exit code 1",
		Severity:  "error",
	}
	pod := kube.PodInfo{
		Name:      "test-pod",
		Namespace: "default",
		Node:      "node-1",
		Phase:     "Running",
	}
	var buf bytes.Buffer
	if err := JSONWithKube(&buf, result, pod); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"kube_pod"`) {
		t.Error("missing kube_pod field")
	}
	if !strings.Contains(out, `"test-pod"`) {
		t.Error("missing pod name in JSON")
	}
	if !strings.Contains(out, `"reason"`) {
		t.Error("missing diagnosis in JSON")
	}
}
