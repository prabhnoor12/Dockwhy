package diagnosis

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/prabhnoor12/dockwhy/internal/docker"
)

// TrendResult summarizes crash patterns across a container's restart history.
type TrendResult struct {
	ContainerName string         `json:"container"`
	Window        string         `json:"window"`
	TotalCrashes  int            `json:"total_crashes"`
	Patterns      []TrendPattern `json:"patterns"`
	FirstCrash    string         `json:"first_crash,omitempty"`
	LastCrash     string         `json:"last_crash,omitempty"`
	AverageGap    string         `json:"average_gap,omitempty"`
	Summary       string         `json:"summary"`
}

// TrendPattern groups crashes by their root cause.
type TrendPattern struct {
	Reason     string `json:"reason"`
	Count      int    `json:"count"`
	Percentage string `json:"percentage"`
	ExitCode   int    `json:"exit_code,omitempty"`
	OOMKilled  bool   `json:"oom_killed,omitempty"`
}

// AnalyzeTrend examines the event history to find crash patterns.
func AnalyzeTrend(containerName string, events []docker.Event, current docker.Container) TrendResult {
	tr := TrendResult{ContainerName: containerName}

	var crashes []crashEvent
	for _, e := range events {
		if e.Action == "die" || e.Action == "kill" || e.Action == "oom" {
			ce := crashEvent{timeNano: e.TimeNano, action: e.Action}
			if code, ok := e.Attrs["exitCode"]; ok {
				ce.exitCode = parseInt(code)
			}
			if oom, ok := e.Attrs["oomKilled"]; ok {
				ce.oomKilled = oom == "true"
			}
			crashes = append(crashes, ce)
		}
	}

	sort.SliceStable(crashes, func(i, j int) bool {
		return crashes[i].timeNano < crashes[j].timeNano
	})

	tr.TotalCrashes = len(crashes)
	if len(crashes) == 0 {
		tr.Summary = fmt.Sprintf("No crashes detected in the event window for %s", containerName)
		return tr
	}

	tr.FirstCrash = formatNanoTime(crashes[0].timeNano)
	tr.LastCrash = formatNanoTime(crashes[len(crashes)-1].timeNano)

	if len(crashes) > 1 {
		totalGap := crashes[len(crashes)-1].timeNano - crashes[0].timeNano
		avgGap := totalGap / int64(len(crashes)-1)
		tr.AverageGap = formatDuration(avgGap)
	}

	groups := make(map[string]*TrendPattern)
	for _, c := range crashes {
		key := trendKey(c)
		if _, ok := groups[key]; !ok {
			groups[key] = &TrendPattern{
				Reason:    trendReason(c),
				ExitCode:  c.exitCode,
				OOMKilled: c.oomKilled,
			}
		}
		groups[key].Count++
	}

	for _, p := range groups {
		p.Percentage = fmt.Sprintf("%.0f%%", float64(p.Count)/float64(len(crashes))*100)
		tr.Patterns = append(tr.Patterns, *p)
	}
	sort.SliceStable(tr.Patterns, func(i, j int) bool {
		return tr.Patterns[i].Count > tr.Patterns[j].Count
	})

	tr.Summary = trendSummary(tr, current)
	return tr
}

type crashEvent struct {
	timeNano  int64
	action    string
	exitCode  int
	oomKilled bool
}

func trendKey(c crashEvent) string {
	if c.oomKilled {
		return "oom"
	}
	return fmt.Sprintf("exit-%d", c.exitCode)
}

func trendReason(c crashEvent) string {
	if c.oomKilled {
		return "out of memory"
	}
	switch c.exitCode {
	case 0:
		return "clean exit"
	case 126:
		return "command not executable"
	case 127:
		return "command not found"
	case 137:
		return "SIGKILL (possible OOM)"
	case 139:
		return "segmentation fault"
	case 143:
		return "SIGTERM (graceful stop)"
	default:
		return fmt.Sprintf("exit code %d", c.exitCode)
	}
}

func trendSummary(tr TrendResult, current docker.Container) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("%d crash(es) in the event window", tr.TotalCrashes))
	if tr.AverageGap != "" {
		parts = append(parts, fmt.Sprintf("average gap: %s", tr.AverageGap))
	}
	if len(tr.Patterns) > 0 {
		top := tr.Patterns[0]
		parts = append(parts, fmt.Sprintf("most common: %s (%s of crashes)", top.Reason, top.Percentage))
	}
	if current.RestartCount > 0 {
		parts = append(parts, fmt.Sprintf("container has restarted %d time(s)", current.RestartCount))
	}
	if isLooping(tr) {
		parts = append(parts, "WARNING: crash loop detected")
	}
	return strings.Join(parts, ", ")
}

func isLooping(tr TrendResult) bool {
	if tr.TotalCrashes < 3 {
		return false
	}
	if tr.AverageGap == "" {
		return false
	}
	avg, err := time.ParseDuration(tr.AverageGap)
	if err != nil {
		return false
	}
	return avg < 5*time.Minute
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func formatNanoTime(nano int64) string {
	if nano <= 0 {
		return "-"
	}
	return time.Unix(0, nano).UTC().Format(time.RFC3339)
}

func formatDuration(nano int64) string {
	d := time.Duration(nano)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
}
