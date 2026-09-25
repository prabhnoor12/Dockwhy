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

package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func fakeKubeCLI(responses map[string]string) *KubeCLI {
	k := NewKubeCLI(context.Background(), 0)
	k.runOverride = func(maxBytes int, args ...string) (commandOutput, error) {
		key := strings.Join(args, " ")
		for pattern, response := range responses {
			if strings.Contains(key, pattern) {
				return commandOutput{stdout: response}, nil
			}
		}
		return commandOutput{}, fmt.Errorf("unexpected kubectl call: %s", key)
	}
	return k
}

const samplePodJSON = `{
  "metadata": {
    "name": "api-pod-abc123",
    "namespace": "production",
    "labels": {"app": "api", "version": "v2"},
    "annotations": {"deployment.kubernetes.io/revision": "3"}
  },
  "spec": {
    "nodeName": "node-1",
    "qosClass": "Burstable"
  },
  "status": {
    "phase": "Running",
    "reason": "",
    "message": "",
    "containerStatuses": [
      {
        "name": "api",
        "image": "api:v2.1.0",
        "containerID": "docker://abc123def456",
        "ready": true,
        "restartCount": 3,
        "state": {"running": {"startedAt": "2026-01-01T00:00:00Z"}},
        "lastState": {
          "terminated": {
            "exitCode": 137,
            "reason": "OOMKilled",
            "oomKilled": true
          }
        }
      },
      {
        "name": "sidecar",
        "image": "envoy:latest",
        "containerID": "docker://789xyz",
        "ready": true,
        "restartCount": 0,
        "state": {"running": {"startedAt": "2026-01-01T00:00:00Z"}}
      }
    ]
  }
}`

func TestGetPod(t *testing.T) {
	k := fakeKubeCLI(map[string]string{
		"get pod api-pod-abc123": samplePodJSON,
	})
	pod, err := k.GetPod("production", "api-pod-abc123")
	if err != nil {
		t.Fatal(err)
	}
	if pod.Name != "api-pod-abc123" {
		t.Errorf("name = %q, want %q", pod.Name, "api-pod-abc123")
	}
	if pod.Namespace != "production" {
		t.Errorf("namespace = %q, want %q", pod.Namespace, "production")
	}
	if pod.Node != "node-1" {
		t.Errorf("node = %q, want %q", pod.Node, "node-1")
	}
	if pod.Phase != "Running" {
		t.Errorf("phase = %q, want %q", pod.Phase, "Running")
	}
	if pod.QoS != "Burstable" {
		t.Errorf("qos = %q, want %q", pod.QoS, "Burstable")
	}
	if pod.Labels["app"] != "api" {
		t.Errorf("label app = %q, want %q", pod.Labels["app"], "api")
	}
	if len(pod.Containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(pod.Containers))
	}
	api := pod.Containers[0]
	if api.Name != "api" {
		t.Errorf("container name = %q, want %q", api.Name, "api")
	}
	if api.RestartCount != 3 {
		t.Errorf("restarts = %d, want 3", api.RestartCount)
	}
	if api.State != "running" {
		t.Errorf("state = %q, want %q", api.State, "running")
	}
	if api.LastExitCode != 137 {
		t.Errorf("last exit code = %d, want 137", api.LastExitCode)
	}
	if !api.LastOOM {
		t.Error("expected last OOM to be true")
	}
	if api.LastReason != "OOMKilled" {
		t.Errorf("last reason = %q, want %q", api.LastReason, "OOMKilled")
	}
	if pod.Restarts != 3 {
		t.Errorf("total restarts = %d, want 3", pod.Restarts)
	}
}

func TestResolveContainer(t *testing.T) {
	k := fakeKubeCLI(map[string]string{
		"get pod api-pod-abc123": samplePodJSON,
	})
	id, err := k.ResolveContainer("api-pod-abc123", "api", "production")
	if err != nil {
		t.Fatal(err)
	}
	if id != "abc123def456" {
		t.Errorf("container ID = %q, want %q", id, "abc123def456")
	}
}

func TestResolveContainerFirstWhenEmpty(t *testing.T) {
	k := fakeKubeCLI(map[string]string{
		"get pod api-pod-abc123": samplePodJSON,
	})
	id, err := k.ResolveContainer("api-pod-abc123", "", "production")
	if err != nil {
		t.Fatal(err)
	}
	if id != "abc123def456" {
		t.Errorf("container ID = %q, want first container", id)
	}
}

func TestResolveContainerNotFound(t *testing.T) {
	k := fakeKubeCLI(map[string]string{
		"get pod api-pod-abc123": samplePodJSON,
	})
	_, err := k.ResolveContainer("api-pod-abc123", "nonexistent", "production")
	if err == nil {
		t.Fatal("expected error for nonexistent container")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestGetPodError(t *testing.T) {
	k := NewKubeCLI(context.Background(), 0)
	k.runOverride = func(maxBytes int, args ...string) (commandOutput, error) {
		return commandOutput{stderr: "Error from server (NotFound): pods \"missing\" not found"}, fmt.Errorf("exit status 1")
	}
	_, err := k.GetPod("default", "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "kubectl") {
		t.Errorf("error should mention kubectl: %v", err)
	}
}

func TestDetectNamespaceDefault(t *testing.T) {
	k := NewKubeCLI(context.Background(), 0)
	ns := k.DetectNamespace()
	if ns != "default" {
		t.Errorf("namespace = %q, want %q (no downward API in test)", ns, "default")
	}
}

func TestPodInfoJSON(t *testing.T) {
	k := fakeKubeCLI(map[string]string{
		"get pod api-pod-abc123": samplePodJSON,
	})
	pod, err := k.GetPod("production", "api-pod-abc123")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(pod)
	if err != nil {
		t.Fatal(err)
	}
	var decoded PodInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != pod.Name || decoded.Namespace != pod.Namespace {
		t.Error("JSON round-trip mismatch")
	}
}
