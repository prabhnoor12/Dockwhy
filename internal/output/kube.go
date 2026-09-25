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
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
	"github.com/prabhnoor12/dockwhy/internal/kube"
)

// KubeContext prints Kubernetes pod metadata as a human-readable header.
func KubeContext(w io.Writer, pod kube.PodInfo) {
	fmt.Fprintf(w, "Kubernetes Pod: %s/%s\n", pod.Namespace, pod.Name)
	if pod.Node != "" {
		fmt.Fprintf(w, "  Node:   %s\n", pod.Node)
	}
	fmt.Fprintf(w, "  Phase:  %s\n", pod.Phase)
	if pod.QoS != "" {
		fmt.Fprintf(w, "  QoS:    %s\n", pod.QoS)
	}
	if pod.Reason != "" {
		fmt.Fprintf(w, "  Reason: %s\n", pod.Reason)
	}
	if pod.Message != "" {
		fmt.Fprintf(w, "  Message: %s\n", pod.Message)
	}
	if len(pod.Labels) > 0 {
		fmt.Fprintf(w, "  Labels: %s\n", formatLabels(pod.Labels))
	}
	if pod.Restarts > 0 {
		fmt.Fprintf(w, "  Total Restarts: %d\n", pod.Restarts)
	}
	if len(pod.Containers) > 0 {
		fmt.Fprintln(w, "  Containers:")
		for _, c := range pod.Containers {
			status := c.State
			if !c.Ready {
				status += " (not ready)"
			}
			restartInfo := ""
			if c.RestartCount > 0 {
				restartInfo = fmt.Sprintf(", %d restart(s)", c.RestartCount)
			}
			lastTerm := ""
			if c.LastReason != "" {
				lastTerm = fmt.Sprintf(", last: %s (exit %d)", c.LastReason, c.LastExitCode)
				if c.LastOOM {
					lastTerm += " [OOM]"
				}
			}
			fmt.Fprintf(w, "    - %s [%s] %s%s%s\n", c.Name, c.Image, status, restartInfo, lastTerm)
		}
	}
}

func formatLabels(labels map[string]string) string {
	var parts []string
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ", ")
}

// JSONWithKube writes the diagnosis as JSON with an additional kube_pod field.
func JSONWithKube(w io.Writer, result diagnosis.Result, pod kube.PodInfo) error {
	wrapped := struct {
		KubePod kube.PodInfo     `json:"kube_pod"`
		Result  diagnosis.Result `json:"result"`
	}{
		KubePod: pod,
		Result:  result,
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(wrapped)
}
