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

func TestReportMarkdownContainsSections(t *testing.T) {
	result := diagnosis.Result{
		Container: docker.Container{
			Name: "api", ID: "abcdef0123456789", Image: "api:latest",
			State:       docker.State{Status: "exited", ExitCode: 137, OOMKilled: true, StartedAt: "2026-09-25T10:00:00Z", FinishedAt: "2026-09-25T10:02:00Z"},
			MemoryLimit: 128 * 1024 * 1024,
		},
		Reason:     "out of memory",
		Summary:    "Docker killed the container after it exceeded its memory limit.",
		Severity:   "critical",
		Confidence: "high",
		Evidence:   []diagnosis.Evidence{{Name: "exit code", Value: "137"}},
		Advice:     []string{"Reduce memory use or raise the limit."},
		Logs:       "fatal: allocation failed",
	}
	var out strings.Builder
	if err := ReportMarkdown(&out, result); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"# Incident Report", "## Summary", "## Diagnosis", "## Evidence", "## Resource Limits", "## Recommended Actions", "## Recent Logs", "out of memory", "api", "137"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in report:\n%s", want, text)
		}
	}
}
