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

package diagnosis

import (
	"testing"

	"github.com/prabhnoor12/dockwhy/internal/docker"
)

func TestAnalyzeResourcesHighMemory(t *testing.T) {
	stats := &docker.Stats{MemoryUsageBytes: 120 * 1024 * 1024, CPUPercent: 10}
	container := docker.Container{MemoryLimit: 128 * 1024 * 1024, NanoCPUs: 1e9}
	report := AnalyzeResources("api", stats, container)
	if len(report.Recommendations) == 0 {
		t.Fatal("expected recommendations")
	}
	memRec := report.Recommendations[0]
	if memRec.Priority != "critical" {
		t.Fatalf("expected critical memory priority, got %q", memRec.Priority)
	}
}

func TestAnalyzeResourcesNoLimit(t *testing.T) {
	stats := &docker.Stats{MemoryUsageBytes: 256 * 1024 * 1024}
	container := docker.Container{}
	report := AnalyzeResources("api", stats, container)
	found := false
	for _, r := range report.Recommendations {
		if r.Resource == "memory" && r.Priority == "warning" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected warning for no memory limit")
	}
}

func TestAnalyzeResourcesNoStats(t *testing.T) {
	report := AnalyzeResources("api", nil, docker.Container{})
	if report.Summary == "" {
		t.Fatal("expected summary")
	}
}

func TestAnalyzeResourcesHealthy(t *testing.T) {
	stats := &docker.Stats{MemoryUsageBytes: 32 * 1024 * 1024, CPUPercent: 5}
	container := docker.Container{MemoryLimit: 256 * 1024 * 1024, NanoCPUs: 2e9}
	report := AnalyzeResources("api", stats, container)
	if report.Summary != "resource limits are appropriate for current usage" {
		t.Fatalf("unexpected summary: %q", report.Summary)
	}
}
