package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
	"github.com/prabhnoor12/dockwhy/internal/docker"
)

func Text(w io.Writer, result diagnosis.Result) error {
	var b strings.Builder
	c := result.Container
	fmt.Fprintf(&b, "Container: %s\n", c.Name)
	fmt.Fprintf(&b, "ID:        %s\n", shortID(c.ID))
	fmt.Fprintf(&b, "Image:     %s\n", c.Image)
	fmt.Fprintf(&b, "State:     %s\n", c.State.Status)
	fmt.Fprintf(&b, "\nDiagnosis [%s, confidence %s]: %s\n%s\n", strings.ToUpper(result.Severity), result.Confidence, result.Reason, result.Summary)
	if len(result.Findings) > 1 {
		fmt.Fprintln(&b, "\nOther findings:")
		for _, finding := range result.Findings[1:] {
			fmt.Fprintf(&b, "  [%s, confidence %s] %s\n", strings.ToUpper(finding.Severity), finding.Confidence, finding.Reason)
			fmt.Fprintf(&b, "    %s\n", finding.Summary)
			for _, evidence := range finding.Evidence {
				fmt.Fprintf(&b, "    evidence: %s=%s\n", evidence.Name, evidence.Value)
			}
		}
	}
	if result.Stats != nil || result.StatsError != "" {
		fmt.Fprintln(&b, "\nCurrent resource usage:")
		if result.StatsError != "" {
			fmt.Fprintf(&b, "  (unavailable: %s)\n", result.StatsError)
		} else {
			stats := result.Stats
			fmt.Fprintf(&b, "  CPU:          %.2f%%\n", stats.CPUPercent)
			if stats.MemoryLimitBytes > 0 {
				fmt.Fprintf(&b, "  Memory:       %s / %s (%.2f%%)\n", formatStatBytes(stats.MemoryUsageBytes), formatStatBytes(stats.MemoryLimitBytes), stats.MemoryPercent)
			} else {
				fmt.Fprintf(&b, "  Memory:       %s (%.2f%%)\n", formatStatBytes(stats.MemoryUsageBytes), stats.MemoryPercent)
			}
			fmt.Fprintf(&b, "  Network I/O:  %s received / %s sent\n", formatStatBytes(stats.NetworkRxBytes), formatStatBytes(stats.NetworkTxBytes))
			fmt.Fprintf(&b, "  Block I/O:    %s read / %s written\n", formatStatBytes(stats.BlockReadBytes), formatStatBytes(stats.BlockWriteBytes))
			fmt.Fprintf(&b, "  Processes:    %d\n", stats.PidsCurrent)
		}
	}

	fmt.Fprintln(&b, "\nDetails:")
	for _, item := range result.Evidence {
		fmt.Fprintf(&b, "  %-20s %s\n", item.Name+":", item.Value)
	}

	if len(result.Advice) > 0 {
		fmt.Fprintln(&b, "\nWhat to check next:")
		for _, advice := range result.Advice {
			fmt.Fprintf(&b, "  - %s\n", advice)
		}
	}

	if len(result.Events) > 0 || result.EventsError != "" {
		fmt.Fprintln(&b, "\nLifecycle timeline:")
		if result.EventsError != "" {
			fmt.Fprintf(&b, "  (unavailable: %s)\n", result.EventsError)
		}
		for _, event := range orderedEvents(result.Events) {
			attrs := formatAttrs(event.Attrs)
			if attrs != "" {
				attrs = " " + attrs
			}
			fmt.Fprintf(&b, "  %-25s %-24s%s\n", formatEventTime(event.TimeNano), describeEvent(event.Action), attrs)
		}
	}

	fmt.Fprintln(&b, "\nRecent logs:")
	if result.LogsSkipped {
		fmt.Fprintln(&b, "  (skipped by --no-logs)")
	} else if result.Logs == "" {
		if result.LogError != "" {
			fmt.Fprintf(&b, "  (unavailable: %s)\n", result.LogError)
		} else {
			fmt.Fprintln(&b, "  (no output)")
		}
	} else {
		for _, line := range strings.Split(result.Logs, "\n") {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}
	if result.LogsTruncated {
		fmt.Fprintln(&b, "  [output truncated]")
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func orderedEvents(events []docker.Event) []docker.Event {
	ordered := append([]docker.Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].TimeNano < ordered[j].TimeNano
	})
	return ordered
}

func formatEventTime(timeNano int64) string {
	if timeNano <= 0 {
		return "-"
	}
	return time.Unix(0, timeNano).UTC().Format("2006-01-02 15:04:05Z07:00")
}

func describeEvent(action string) string {
	switch action {
	case "create":
		return "created"
	case "start":
		return "started"
	case "stop":
		return "stopped"
	case "kill":
		return "killed"
	case "die":
		return "exited"
	case "restart":
		return "restarted"
	case "destroy":
		return "destroyed"
	}
	if strings.HasPrefix(action, "health_status:") {
		return "health check: " + strings.TrimSpace(strings.TrimPrefix(action, "health_status:"))
	}
	return action
}

func formatStatBytes(value int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	n := float64(value)
	i := 0
	for n >= 1024 && i < len(units)-1 {
		n /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d %s", value, units[i])
	}
	return fmt.Sprintf("%.1f %s", n, units[i])
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func formatAttrs(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+attrs[key])
	}
	return strings.Join(parts, " ")
}
