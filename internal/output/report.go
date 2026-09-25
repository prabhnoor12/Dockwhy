package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
	"github.com/prabhnoor12/dockwhy/internal/format"
)

// ReportMarkdown writes a complete incident report in Markdown format.
func ReportMarkdown(w io.Writer, result diagnosis.Result) error {
	var b strings.Builder
	c := result.Container
	now := time.Now().UTC().Format(time.RFC3339)

	b.WriteString("# Incident Report\n\n")
	fmt.Fprintf(&b, "**Generated:** %s  \n", now)
	fmt.Fprintf(&b,("**Tool:** dockwhy  \n"))
	b.WriteString("\n---\n\n")

	b.WriteString("## Summary\n\n")
	fmt.Fprintf(&b, "| Field | Value |\n")
	fmt.Fprintf(&b, "|-------|-------|\n")
	fmt.Fprintf(&b, "| Container | %s |\n", c.Name)
	fmt.Fprintf(&b, "| ID | %s |\n", shortID(c.ID))
	fmt.Fprintf(&b, "| Image | %s |\n", c.Image)
	fmt.Fprintf(&b, "| State | %s |\n", c.State.Status)
	fmt.Fprintf(&b, "| Exit Code | %d |\n", c.State.ExitCode)
	if c.State.OOMKilled {
		fmt.Fprintf(&b, "| OOM Killed | Yes |\n")
	}
	if c.State.StartedAt != "" {
		fmt.Fprintf(&b, "| Started | %s |\n", c.State.StartedAt)
	}
	if c.State.FinishedAt != "" {
		fmt.Fprintf(&b, "| Finished | %s |\n", c.State.FinishedAt)
	}
	if c.ComposeProject != "" {
		fmt.Fprintf(&b, "| Compose Project | %s |\n", c.ComposeProject)
	}
	if c.ComposeService != "" {
		fmt.Fprintf(&b, "| Compose Service | %s |\n", c.ComposeService)
	}
	b.WriteString("\n")

	b.WriteString("## Diagnosis\n\n")
	fmt.Fprintf(&b, "**Severity:** %s  \n", strings.ToUpper(result.Severity))
	fmt.Fprintf(&b, "**Confidence:** %s  \n", result.Confidence)
	fmt.Fprintf(&b, "**Reason:** %s  \n", result.Reason)
	fmt.Fprintf(&b, "\n%s\n\n", result.Summary)

	if len(result.Findings) > 1 {
		b.WriteString("### Additional Findings\n\n")
		for _, f := range result.Findings[1:] {
			fmt.Fprintf(&b, "%d. **%s** [%s, confidence %s]: %s\n", f.Rank, f.Reason, strings.ToUpper(f.Severity), f.Confidence, f.Summary)
		}
		b.WriteString("\n")
	}

	if len(result.Evidence) > 0 {
		b.WriteString("## Evidence\n\n")
		b.WriteString("| Key | Value |\n")
		b.WriteString("|-----|-------|\n")
		for _, e := range result.Evidence {
			fmt.Fprintf(&b, "| %s | %s |\n", e.Name, e.Value)
		}
		b.WriteString("\n")
	}

	if c.MemoryLimit > 0 || c.NanoCPUs > 0 || c.PidsLimit > 0 {
		b.WriteString("## Resource Limits\n\n")
		b.WriteString("| Resource | Limit |\n")
		b.WriteString("|----------|-------|\n")
		if c.MemoryLimit > 0 {
			fmt.Fprintf(&b, "| Memory | %s |\n", format.Bytes(c.MemoryLimit))
		}
		if c.NanoCPUs > 0 {
			fmt.Fprintf(&b, "| CPU | %.2f cores |\n", float64(c.NanoCPUs)/1e9)
		}
		if c.PidsLimit > 0 {
			fmt.Fprintf(&b, "| PIDs | %d |\n", c.PidsLimit)
		}
		if c.DiskLimit != "" {
			fmt.Fprintf(&b, "| Disk | %s |\n", c.DiskLimit)
		}
		b.WriteString("\n")
	}

	if result.Stats != nil {
		b.WriteString("## Resource Usage (at time of diagnosis)\n\n")
		s := result.Stats
		b.WriteString("| Metric | Value |\n")
		b.WriteString("|--------|-------|\n")
		fmt.Fprintf(&b, "| CPU | %.2f%% |\n", s.CPUPercent)
		fmt.Fprintf(&b, "| Memory | %s |\n", format.Bytes(s.MemoryUsageBytes))
		if s.MemoryLimitBytes > 0 {
			fmt.Fprintf(&b, "| Memory Limit | %s (%.1f%%) |\n", format.Bytes(s.MemoryLimitBytes), s.MemoryPercent)
		}
		fmt.Fprintf(&b, "| Network I/O | %s rx / %s tx |\n", format.Bytes(s.NetworkRxBytes), format.Bytes(s.NetworkTxBytes))
		fmt.Fprintf(&b, "| Block I/O | %s read / %s written |\n", format.Bytes(s.BlockReadBytes), format.Bytes(s.BlockWriteBytes))
		fmt.Fprintf(&b, "| Processes | %d |\n", s.PidsCurrent)
		b.WriteString("\n")
	}

	if len(result.Advice) > 0 {
		b.WriteString("## Recommended Actions\n\n")
		for i, advice := range result.Advice {
			fmt.Fprintf(&b, "%d. %s\n", i+1, advice)
		}
		b.WriteString("\n")
	}

	if len(result.Events) > 0 {
		b.WriteString("## Lifecycle Timeline\n\n")
		b.WriteString("| Time | Event | Details |\n")
		b.WriteString("|------|-------|---------|\n")
		for _, e := range result.Events {
			attrs := formatAttrs(e.Attrs)
			fmt.Fprintf(&b, "| %s | %s | %s |\n", formatEventTime(e.TimeNano), describeEvent(e.Action), attrs)
		}
		b.WriteString("\n")
	}

	if result.Logs != "" {
		b.WriteString("## Recent Logs\n\n")
		b.WriteString("```\n")
		b.WriteString(result.Logs)
		b.WriteString("\n```\n\n")
	}

	if result.EventsError != "" {
		fmt.Fprintf(&b, "> **Note:** Events were unavailable: %s\n\n", result.EventsError)
	}
	if result.LogError != "" {
		fmt.Fprintf(&b, "> **Note:** Logs were unavailable: %s\n\n", result.LogError)
	}

	_, err := io.WriteString(w, b.String())
	return err
}
