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

func TestAnalyzeTrendGroupsCrashes(t *testing.T) {
	events := []docker.Event{
		{TimeNano: 1000, Action: "die", Attrs: map[string]string{"exitCode": "137"}},
		{TimeNano: 2000, Action: "die", Attrs: map[string]string{"exitCode": "137"}},
		{TimeNano: 3000, Action: "die", Attrs: map[string]string{"exitCode": "1"}},
	}
	result := AnalyzeTrend("api", events, docker.Container{RestartCount: 3})
	if result.TotalCrashes != 3 {
		t.Fatalf("total crashes = %d, want 3", result.TotalCrashes)
	}
	if len(result.Patterns) != 2 {
		t.Fatalf("patterns = %d, want 2", len(result.Patterns))
	}
	if result.Patterns[0].Count != 2 {
		t.Fatalf("top pattern count = %d, want 2", result.Patterns[0].Count)
	}
}

func TestAnalyzeTrendDetectsCrashLoop(t *testing.T) {
	events := []docker.Event{
		{TimeNano: 1000, Action: "die", Attrs: map[string]string{"exitCode": "1"}},
		{TimeNano: 2000000000, Action: "die", Attrs: map[string]string{"exitCode": "1"}},
		{TimeNano: 3000000000, Action: "die", Attrs: map[string]string{"exitCode": "1"}},
		{TimeNano: 4000000000, Action: "die", Attrs: map[string]string{"exitCode": "1"}},
	}
	result := AnalyzeTrend("api", events, docker.Container{})
	if result.TotalCrashes != 4 {
		t.Fatalf("total = %d", result.TotalCrashes)
	}
	if result.FirstCrash == "" || result.LastCrash == "" {
		t.Fatal("expected first/last crash timestamps")
	}
}

func TestAnalyzeTrendNoCrashes(t *testing.T) {
	result := AnalyzeTrend("api", nil, docker.Container{})
	if result.TotalCrashes != 0 {
		t.Fatalf("total = %d", result.TotalCrashes)
	}
}
