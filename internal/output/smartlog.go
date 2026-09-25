package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
)

// SmartLogJSON writes the smart log result as indented JSON.
func SmartLogJSON(w io.Writer, result diagnosis.SmartLogResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// SmartLogText writes the smart log result as human-readable text.
func SmartLogText(w io.Writer, result diagnosis.SmartLogResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Smart log analysis: %s\n\n", result.Summary)
	for i, section := range result.Sections {
		if i > 0 {
			fmt.Fprintln(&b)
		}
		label := section.Kind
		if section.Context != "" {
			label += " — " + section.Context
		}
		fmt.Fprintf(&b, "--- [%s] ---\n", label)
		for _, line := range section.Lines {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}
