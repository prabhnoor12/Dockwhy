package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/dockwhy/dockwhy/internal/diagnosis"
)

func Text(w io.Writer, result diagnosis.Result) error {
	var b strings.Builder
	c := result.Container
	fmt.Fprintf(&b, "Container: %s\n", c.Name)
	fmt.Fprintf(&b, "ID:        %s\n", shortID(c.ID))
	fmt.Fprintf(&b, "Image:     %s\n", c.Image)
	fmt.Fprintf(&b, "State:     %s\n", c.State.Status)
	fmt.Fprintf(&b, "\nDiagnosis [%s, confidence %s]: %s\n%s\n", strings.ToUpper(result.Severity), result.Confidence, result.Reason, result.Summary)

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
		fmt.Fprintln(&b, "\nDocker events:")
		if result.EventsError != "" {
			fmt.Fprintf(&b, "  (unavailable: %s)\n", result.EventsError)
		}
		for _, event := range result.Events {
			fmt.Fprintf(&b, "  %d  %-16s %s\n", event.TimeNano, event.Action, formatAttrs(event.Attrs))
		}
	}

	fmt.Fprintln(&b, "\nRecent logs:")
	if result.Logs == "" {
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
