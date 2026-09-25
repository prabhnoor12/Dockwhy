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

// PodInfo holds the Kubernetes metadata for a pod that is relevant to
// container diagnosis.
type PodInfo struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Node        string            `json:"node"`
	Phase       string            `json:"phase"`
	Reason      string            `json:"reason,omitempty"`
	Message     string            `json:"message,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Containers  []PodContainer    `json:"containers"`
	QoS         string            `json:"qos,omitempty"`
	Restarts    int               `json:"restarts"`
}

// PodContainer is one container within a pod.
type PodContainer struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	ContainerID  string `json:"container_id"`
	Ready        bool   `json:"ready"`
	RestartCount int    `json:"restart_count"`
	State        string `json:"state"`
	LastExitCode int    `json:"last_exit_code,omitempty"`
	LastReason   string `json:"last_reason,omitempty"`
	LastOOM      bool   `json:"last_oom,omitempty"`
}

// Client abstracts Kubernetes CLI access for testing.
type Client interface {
	GetPod(namespace, name string) (PodInfo, error)
	ResolveContainer(podName, containerName, namespace string) (containerID string, err error)
	DetectNamespace() string
}
