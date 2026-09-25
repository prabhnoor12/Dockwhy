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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// KubeCLI talks to Kubernetes using kubectl, the same CLI users use
// interactively. It respects KUBECONFIG and the active context.
type KubeCLI struct {
	Binary  string
	Timeout time.Duration
	ctx     context.Context
	runOverride func(maxBytes int, args ...string) (commandOutput, error)
}

// NewKubeCLI creates a client that shells out to kubectl.
func NewKubeCLI(ctx context.Context, timeout time.Duration) *KubeCLI {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return &KubeCLI{Binary: "kubectl", Timeout: timeout, ctx: ctx}
}

// DetectNamespace returns the namespace from the downward API if running
// inside a pod, or "default" otherwise.
func (k *KubeCLI) DetectNamespace() string {
	if ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if trimmed := strings.TrimSpace(string(ns)); trimmed != "" {
			return trimmed
		}
	}
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	return "default"
}

// GetPod fetches pod metadata and container statuses.
func (k *KubeCLI) GetPod(namespace, name string) (PodInfo, error) {
	out, err := k.run(4<<20, "get", "pod", name, "-n", namespace, "-o", "json")
	if err != nil {
		return PodInfo{}, kubeCommandError("get pod", name, out, err)
	}
	var raw podJSON
	if err := json.Unmarshal([]byte(out.stdout), &raw); err != nil {
		return PodInfo{}, fmt.Errorf("decode kubectl output: %w", err)
	}
	return raw.podInfo(), nil
}

// ResolveContainer finds the Docker container ID for a given pod and
// container name so that the existing Docker diagnosis path can be used.
func (k *KubeCLI) ResolveContainer(podName, containerName, namespace string) (string, error) {
	pod, err := k.GetPod(namespace, podName)
	if err != nil {
		return "", err
	}
	for _, c := range pod.Containers {
		if c.Name == containerName || containerName == "" {
			id := c.ContainerID
			if idx := strings.Index(id, "://"); idx >= 0 {
				id = id[idx+3:]
			}
			if id == "" {
				return "", fmt.Errorf("container %q in pod %q has no container ID (state: %s)", c.Name, podName, c.State)
			}
			return id, nil
		}
	}
	return "", fmt.Errorf("container %q not found in pod %q", containerName, podName)
}

type commandOutput struct {
	stdout    string
	stderr    string
	truncated bool
}

func (k *KubeCLI) run(maxBytes int, args ...string) (commandOutput, error) {
	if k.runOverride != nil {
		return k.runOverride(maxBytes, args...)
	}
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	timeout := k.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(k.ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, k.Binary, args...)
	var stdout, stderr limitedBuffer
	stdout.limit, stderr.limit = maxBytes, 1<<20
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return commandOutput{stdout: stdout.String(), stderr: stderr.String(), truncated: stdout.truncated || stderr.truncated}, ctx.Err()
	}
	return commandOutput{stdout: stdout.String(), stderr: stderr.String(), truncated: stdout.truncated || stderr.truncated}, err
}

func kubeCommandError(action, name string, output commandOutput, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("kubectl %s %q timed out: %w", action, name, err)
	}
	detail := strings.TrimSpace(output.stderr)
	if detail == "" {
		detail = strings.TrimSpace(output.stdout)
	}
	if detail == "" {
		detail = err.Error()
	}
	if output.truncated {
		detail += " [kubectl output truncated]"
	}
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("kubectl is not installed or not on PATH: %w", err)
	}
	return fmt.Errorf("kubectl %s %q failed: %s: %w", action, name, detail, err)
}

type limitedBuffer struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.Buffer.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

type podJSON struct {
	Metadata struct {
		Name        string            `json:"name"`
		Namespace   string            `json:"namespace"`
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		NodeName string `json:"nodeName"`
		QOSClass string `json:"qosClass"`
	} `json:"spec"`
	Status struct {
		Phase            string            `json:"phase"`
		Reason           string            `json:"reason"`
		Message          string            `json:"message"`
		ContainerStatuses []containerStatus `json:"containerStatuses"`
	} `json:"status"`
}

type containerStatus struct {
	Name                 string `json:"name"`
	Image                string `json:"image"`
	ContainerID          string `json:"containerID"`
	Ready                bool   `json:"ready"`
	RestartCount         int    `json:"restartCount"`
	State                map[string]json.RawMessage `json:"state"`
	LastTerminationState map[string]json.RawMessage `json:"lastState"`
}

func (p podJSON) podInfo() PodInfo {
	info := PodInfo{
		Name:        p.Metadata.Name,
		Namespace:   p.Metadata.Namespace,
		Node:        p.Spec.NodeName,
		Phase:       p.Status.Phase,
		Reason:      p.Status.Reason,
		Message:     p.Status.Message,
		Labels:      p.Metadata.Labels,
		Annotations: p.Metadata.Annotations,
		QoS:         p.Spec.QOSClass,
	}
	totalRestarts := 0
	for _, cs := range p.Status.ContainerStatuses {
		pc := PodContainer{
			Name:         cs.Name,
			Image:        cs.Image,
			ContainerID:  cs.ContainerID,
			Ready:        cs.Ready,
			RestartCount: cs.RestartCount,
		}
		totalRestarts += cs.RestartCount
		pc.State, pc.LastExitCode, pc.LastReason, pc.LastOOM = parseContainerState(cs)
		info.Containers = append(info.Containers, pc)
	}
	info.Restarts = totalRestarts
	return info
}

func parseContainerState(cs containerStatus) (state string, lastExitCode int, lastReason string, lastOOM bool) {
	for k := range cs.State {
		state = k
		break
	}
	if raw, ok := cs.LastTerminationState["terminated"]; ok {
		var t struct {
			ExitCode int    `json:"exitCode"`
			Reason   string `json:"reason"`
			OOMKilled bool  `json:"oomKilled"`
		}
		if err := json.Unmarshal(raw, &t); err == nil {
			lastExitCode = t.ExitCode
			lastReason = t.Reason
			lastOOM = t.OOMKilled
		}
	}
	return
}
