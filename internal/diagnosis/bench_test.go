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
	"strings"
	"testing"

	"github.com/prabhnoor12/dockwhy/internal/docker"
)

func BenchmarkAnalyzeDetailed(b *testing.B) {
	c := docker.Container{
		Name:  "bench-api",
		ID:    "abc123def456",
		Image: "api:v2.1",
		State: docker.State{
			Status:     "exited",
			ExitCode:   137,
			OOMKilled:  true,
			StartedAt:  "2025-01-01T00:00:00Z",
			FinishedAt: "2025-01-01T00:05:00Z",
			Health:     &docker.Health{Status: "unhealthy", FailingStreak: 5},
		},
		RestartCount: 3,
		MemoryLimit:  256 * 1024 * 1024,
		Labels:       map[string]string{"app": "api", "env": "prod"},
	}
	logs := "2025-01-01 panic: runtime error\nFATAL: out of memory\nconnection refused\n"
	events := []docker.Event{
		{TimeNano: 100, Action: "start"},
		{TimeNano: 200, Action: "oom"},
		{TimeNano: 300, Action: "die"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AnalyzeDetailed(c, logs, false, events, nil, nil)
	}
}

func BenchmarkAnalyzeDetailedMinimal(b *testing.B) {
	c := docker.Container{
		Name:  "simple",
		State: docker.State{Status: "exited", ExitCode: 0},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AnalyzeDetailed(c, "", false, nil, nil, nil)
	}
}

func BenchmarkExtractSmartLogs(b *testing.B) {
	lines := make([]string, 500)
	for i := range lines {
		lines[i] = "2025-01-01 INFO normal operation line"
	}
	lines[100] = "panic: runtime error: index out of range"
	lines[101] = "goroutine 1 [running]:"
	lines[250] = "FATAL: configuration missing"
	lines[400] = "dial tcp 127.0.0.1:5432: connection refused"
	logs := strings.Join(lines, "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExtractSmartLogs(logs)
	}
}

func BenchmarkApplyRules(b *testing.B) {
	code := 137
	rules := []Rule{
		{Name: "rule1", Reason: "test", Match: RuleMatch{ContainerGlob: "api-*"}},
		{Name: "rule2", Reason: "test", Match: RuleMatch{LogPattern: "error"}},
		{Name: "rule3", Reason: "test", Match: RuleMatch{ExitCode: &code}},
	}
	c := docker.Container{Name: "api-v2", State: docker.State{ExitCode: 137}}
	logs := "error: something failed"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ApplyRules(rules, c, logs)
	}
}
