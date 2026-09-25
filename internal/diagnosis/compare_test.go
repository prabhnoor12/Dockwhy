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

func TestCompareContainersIdentical(t *testing.T) {
	c := docker.Container{
		Name:  "test",
		Image: "api:latest",
		State: docker.State{Status: "running"},
	}
	result := CompareContainers(c, c)
	if len(result.Diffs) != 0 {
		t.Errorf("expected no diffs for identical containers, got %d", len(result.Diffs))
	}
	if result.CriticalCount != 0 || result.WarningCount != 0 {
		t.Error("expected zero counts for identical containers")
	}
}

func TestCompareContainersImageDiff(t *testing.T) {
	a := docker.Container{Name: "a", Image: "api:v1"}
	b := docker.Container{Name: "b", Image: "api:v2"}
	result := CompareContainers(a, b)
	if len(result.Diffs) == 0 {
		t.Fatal("expected diffs")
	}
	found := false
	for _, d := range result.Diffs {
		if d.Field == "image" {
			found = true
			if d.Before != "api:v1" || d.After != "api:v2" {
				t.Errorf("image diff = %q -> %q", d.Before, d.After)
			}
			if d.Category != DiffCritical {
				t.Errorf("image diff should be critical, got %s", d.Category)
			}
		}
	}
	if !found {
		t.Error("missing image diff")
	}
	if result.CriticalCount < 1 {
		t.Error("expected at least 1 critical diff")
	}
}

func TestCompareContainersMemoryDiff(t *testing.T) {
	a := docker.Container{Name: "a", MemoryLimit: 256 << 20}
	b := docker.Container{Name: "b", MemoryLimit: 512 << 20}
	result := CompareContainers(a, b)
	found := false
	for _, d := range result.Diffs {
		if d.Field == "memory_limit" {
			found = true
			if d.Category != DiffCritical {
				t.Errorf("memory_limit diff should be critical, got %s", d.Category)
			}
		}
	}
	if !found {
		t.Error("missing memory_limit diff")
	}
}

func TestCompareContainersMultipleDiffs(t *testing.T) {
	a := docker.Container{
		Name:        "a",
		Image:       "api:v1",
		User:        "root",
		MemoryLimit: 256 << 20,
		PidsLimit:   100,
	}
	b := docker.Container{
		Name:        "b",
		Image:       "api:v2",
		User:        "app",
		MemoryLimit: 512 << 20,
		PidsLimit:   200,
	}
	result := CompareContainers(a, b)
	if len(result.Diffs) < 4 {
		t.Errorf("expected at least 4 diffs, got %d", len(result.Diffs))
	}
	if result.Summary == "" {
		t.Error("summary should not be empty")
	}
}

func TestCompareContainersMountDiff(t *testing.T) {
	a := docker.Container{
		Name: "a",
		Mounts: []docker.Mount{
			{Source: "/data", Destination: "/app/data"},
		},
	}
	b := docker.Container{
		Name:   "b",
		Mounts: nil,
	}
	result := CompareContainers(a, b)
	found := false
	for _, d := range result.Diffs {
		if d.Field == "mounts" {
			found = true
		}
	}
	if !found {
		t.Error("missing mounts diff")
	}
}

func TestCompareSummaryNoDiffs(t *testing.T) {
	c := docker.Container{Name: "same", Image: "img"}
	result := CompareContainers(c, c)
	if result.Summary == "" {
		t.Error("summary should not be empty")
	}
}
