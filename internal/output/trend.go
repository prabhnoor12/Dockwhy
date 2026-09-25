package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
)

// TrendJSON writes the trend result as indented JSON.
func TrendJSON(w io.Writer, result diagnosis.TrendResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// TrendText writes the trend result as human-readable text.
func TrendText(w io.Writer, result diagnosis.TrendResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Trend analysis: %s\n", result.ContainerName)
	fmt.Fprintf(&b, "Total crashes: %d\n", result.TotalCrashes)
	if result.FirstCrash != "" {
		fmt.Fprintf(&b, "First crash:   %s\n", result.FirstCrash)
	}
	if result.LastCrash != "" {
		fmt.Fprintf(&b, "Last crash:    %s\n", result.LastCrash)
	}
	if result.AverageGap != "" {
		fmt.Fprintf(&b, "Average gap:   %s\n", result.AverageGap)
	}
	fmt.Fprintf(&b, "\n%s\n", result.Summary)

	if len(result.Patterns) > 0 {
		fmt.Fprintln(&b, "\nCrash patterns:")
		for _, p := range result.Patterns {
			marker := ""
			if p.OOMKilled {
				marker = " [OOM]"
			}
			fmt.Fprintf(&b, "  %-30s %d time(s) (%s)%s\n", p.Reason, p.Count, p.Percentage, marker)
		}
	}

	_, err := io.WriteString(w, b.String())
	return err
}
