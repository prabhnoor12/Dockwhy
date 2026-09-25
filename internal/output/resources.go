package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
)

// ResourceJSON writes the resource report as indented JSON.
func ResourceJSON(w io.Writer, report diagnosis.ResourceReport) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// ResourceText writes the resource report as human-readable text.
func ResourceText(w io.Writer, report diagnosis.ResourceReport) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Resource analysis: %s\n", report.Container)
	fmt.Fprintf(&b, "%s\n", report.Summary)

	for _, rec := range report.Recommendations {
		fmt.Fprintln(&b)
		priority := strings.ToUpper(rec.Priority)
		fmt.Fprintf(&b, "  [%s] %s\n", priority, rec.Resource)
		fmt.Fprintf(&b, "    Current:  %s\n", rec.Current)
		if rec.Peak != "" {
			fmt.Fprintf(&b, "    Peak:     %s\n", rec.Peak)
		}
		fmt.Fprintf(&b, "    Usage:    %s\n", rec.Utilization)
		fmt.Fprintf(&b, "    Advice:   %s\n", rec.Suggestion)
	}

	_, err := io.WriteString(w, b.String())
	return err
}
