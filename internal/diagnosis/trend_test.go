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
