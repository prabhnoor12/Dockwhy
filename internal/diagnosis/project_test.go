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

func TestAnalyzeProjectIdentifiesRootCause(t *testing.T) {
	db := docker.Container{
		Name: "shop-db-1", ComposeService: "db",
		State: docker.State{Status: "exited", ExitCode: 137, OOMKilled: true, FinishedAt: "2026-09-25T10:00:00Z"},
	}
	api := docker.Container{
		Name: "shop-api-1", ComposeService: "api",
		State: docker.State{Status: "exited", ExitCode: 1, FinishedAt: "2026-09-25T10:00:30Z"},
	}
	diagnoses := []ContainerDiagnosis{
		{Container: db, Result: Analyze(db, "", nil)},
		{Container: api, Result: Analyze(api, "connection refused", nil)},
	}
	events := map[string][]docker.Event{
		"shop-db-1":  {{TimeNano: 1000, Action: "die", Attrs: map[string]string{"exitCode": "137"}}},
		"shop-api-1": {{TimeNano: 1030, Action: "die", Attrs: map[string]string{"exitCode": "1"}}},
	}
	result := AnalyzeProject("shop", diagnoses, events)
	if result.RootCause == nil {
		t.Fatal("expected root cause")
	}
	if result.RootCause.Container.Name != "shop-db-1" {
		t.Fatalf("root cause = %q, want shop-db-1", result.RootCause.Container.Name)
	}
	if len(result.Timeline) != 2 {
		t.Fatalf("expected 2 timeline entries, got %d", len(result.Timeline))
	}
	if result.Timeline[0].Container != "shop-db-1" {
		t.Fatal("timeline not sorted chronologically")
	}
}

func TestAnalyzeProjectNoFailures(t *testing.T) {
	api := docker.Container{
		Name: "shop-api-1", ComposeService: "api",
		State: docker.State{Status: "running", Running: true},
	}
	diagnoses := []ContainerDiagnosis{
		{Container: api, Result: Analyze(api, "", nil)},
	}
	result := AnalyzeProject("shop", diagnoses, nil)
	if result.RootCause != nil {
		t.Fatalf("expected no root cause, got %v", result.RootCause.Container.Name)
	}
}

func TestProjectSummaryFormat(t *testing.T) {
	pr := ProjectResult{
		Project: "shop",
		Containers: []ContainerDiagnosis{
			{Container: docker.Container{Name: "api", State: docker.State{Status: "running", Running: true}}},
			{Container: docker.Container{Name: "db", State: docker.State{Status: "exited", ExitCode: 1}}},
		},
		RootCause: &ContainerDiagnosis{
			Container: docker.Container{Name: "db"},
			Result:    Result{Reason: "disk full"},
		},
	}
	summary := projectSummary(pr)
	if summary == "" {
		t.Fatal("summary is empty")
	}
	if !containsStr(summary, "shop") || !containsStr(summary, "root cause") || !containsStr(summary, "db") {
		t.Fatalf("summary missing expected content: %s", summary)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
