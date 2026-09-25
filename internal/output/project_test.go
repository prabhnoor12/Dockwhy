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
	"strings"
	"testing"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
	"github.com/prabhnoor12/dockwhy/internal/docker"
)

func TestProjectTextShowsRootCause(t *testing.T) {
	result := diagnosis.ProjectResult{
		Project: "shop",
		Containers: []diagnosis.ContainerDiagnosis{
			{Container: docker.Container{Name: "shop-db-1", ComposeService: "db", State: docker.State{Status: "exited", ExitCode: 137}}, Result: diagnosis.Result{Reason: "out of memory", Severity: "critical", Summary: "memory limit exceeded", Advice: []string{"raise the limit"}}},
			{Container: docker.Container{Name: "shop-api-1", ComposeService: "api", State: docker.State{Status: "exited", ExitCode: 1}}, Result: diagnosis.Result{Reason: "application exited", Severity: "info", Summary: "process exited"}},
		},
		RootCause: &diagnosis.ContainerDiagnosis{
			Container: docker.Container{Name: "shop-db-1"},
			Result:    diagnosis.Result{Reason: "out of memory", Summary: "memory limit exceeded", Advice: []string{"raise the limit"}},
		},
		Summary: "2 containers, root cause: shop-db-1",
		Timeline: []diagnosis.TimelineEntry{
			{TimeNano: 1000, Container: "shop-db-1", Service: "db", Event: "die", Detail: "exit 137"},
		},
	}
	var out strings.Builder
	if err := ProjectText(&out, result); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"shop", "Root cause:", "shop-db-1", "out of memory", "Timeline:", "raise the limit", "->"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in output:\n%s", want, text)
		}
	}
}

func TestProjectJSONContainsRootCause(t *testing.T) {
	result := diagnosis.ProjectResult{
		Project: "shop",
		RootCause: &diagnosis.ContainerDiagnosis{
			Container: docker.Container{Name: "db"},
			Result:    diagnosis.Result{Reason: "disk full"},
		},
	}
	var out strings.Builder
	if err := ProjectJSON(&out, result); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"root_cause"`) || !strings.Contains(out.String(), `"disk full"`) {
		t.Fatalf("unexpected JSON: %s", out.String())
	}
}
